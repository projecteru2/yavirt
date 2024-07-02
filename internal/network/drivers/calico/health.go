// Copyright (c) 2016 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package calico

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"reflect"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/utils"
	"github.com/samber/lo"
	"github.com/shirou/gopsutil/process"
)

// CheckNodeStatus prints status of the node and returns error (if any)
func CheckNodeStatus() error {
	// Go through running processes and check if `calico-felix` processes is not running
	processes, err := process.Processes()
	if err != nil {
		return err
	}

	// For older versions of calico/node, the process was called `calico-felix`. Newer ones use `calico-node -felix`.
	if !utils.PSContains([]string{"calico-felix"}, processes) && !utils.PSContains([]string{"calico-node", "-felix"}, processes) {
		// Return and print message if calico-node is not running
		return errors.New("calico process is not running")
	}

	if !utils.PSContains([]string{"bird"}, processes) {
		return errors.New("BIRDv4 process: 'bird' is not running")
	}
	peers, err := getBIRDPeers("4")
	if err != nil {
		return err
	}
	if checkActivePeers(peers) != nil {
		return errors.New("No active IPV4 peers")
	}
	// // Check if birdv6 process is running, print the BGP peer table if it is, else print a warning
	// if psContains([]string{"bird6"}, processes) {
	// 	if peers, err := getBIRDPeers("6"); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	fmt.Printf("\nINFO: BIRDv6 process: 'bird6' is not running.\n")
	// }

	return nil
}

// Check for Word_<IP> where every octate is separated by "_", regardless of IP protocols
// Example match: "Mesh_192_168_56_101" or "Mesh_fd80_24e2_f998_72d7__2"
var bgpPeerRegex = regexp.MustCompile(`^(Global|Node|Mesh)_(.+)$`)

// Mapping the BIRD/GoBGP type extracted from the peer name to the display type.
var bgpTypeMap = map[string]string{
	"Global": "global",
	"Mesh":   "node-to-node mesh",
	"Node":   "node specific",
}

// Timeout for querying BIRD
var birdTimeOut = 2 * time.Second

// Expected BIRD protocol table columns
var birdExpectedHeadings = []string{"name", "proto", "table", "state", "since", "info"}

// bgpPeer is a structure containing details about a BGP peer.
type bgpPeer struct {
	PeerIP   string
	PeerType string
	State    string
	Since    string
	BGPState string
	Info     string
}

// Unmarshal a peer from a line in the BIRD protocol output.  Returns true if
// successful, false otherwise.
func (b *bgpPeer) unmarshalBIRD(line, ipSep string) bool {
	// Split into fields.  We expect at least 6 columns:
	// 	name, proto, table, state, since and info.
	// The info column contains the BGP state plus possibly some additional
	// info (which will be columns > 6).
	//
	// Peer names will be of the format described by bgpPeerRegex.
	log.Debugf(context.TODO(), "Parsing line: %s", line)
	columns := strings.Fields(line)
	if len(columns) < 6 {
		log.Debug(context.TODO(), "Not a valid line: fewer than 6 columns")
		return false
	}
	if columns[1] != "BGP" {
		log.Debug(context.TODO(), "Not a valid line: protocol is not BGP")
		return false
	}

	// Check the name of the peer is of the correct format.  This regex
	// returns two components:
	// -  A type (Global|Node|Mesh) which we can map to a display type
	// -  An IP address (with _ separating the octets)
	sm := bgpPeerRegex.FindStringSubmatch(columns[0])
	if len(sm) != 3 {
		log.Debugf(context.TODO(), "Not a valid line: peer name '%s' is not correct format", columns[0])
		return false
	}
	var ok bool
	b.PeerIP = strings.ReplaceAll(sm[2], "_", ipSep)
	if b.PeerType, ok = bgpTypeMap[sm[1]]; !ok {
		log.Debugf(context.TODO(), "Not a valid line: peer type '%s' is not recognized", sm[1])
		return false
	}

	// Store remaining columns (piecing back together the info string)
	b.State = columns[3]
	b.Since = columns[4]
	b.BGPState = columns[5]
	if len(columns) > 6 {
		b.Info = strings.Join(columns[6:], " ")
	}

	return true
}

// getBIRDPeers queries BIRD and displays the local peers in table format.
func getBIRDPeers(ipv string) ([]bgpPeer, error) {
	log.Debugf(context.TODO(), "Print BIRD peers for IPv%s", ipv)
	birdSuffix := ""
	if ipv == "6" {
		birdSuffix = "6"
	}

	log.Debugf(context.TODO(), "IPv%s BGP status", ipv)

	// Try connecting to the bird socket in `/var/run/calico/` first to get the data
	c, err := net.Dial("unix", fmt.Sprintf("/var/run/calico/bird%s.ctl", birdSuffix))
	if err != nil {
		// If that fails, try connecting to bird socket in `/var/run/bird` (which is the
		// default socket location for bird install) for non-containerized installs
		log.Debug(context.TODO(), "Failed to connect to BIRD socket in /var/run/calic, trying /var/run/bird \n")
		c, err = net.Dial("unix", fmt.Sprintf("/var/run/bird/bird%s.ctl", birdSuffix))
		if err != nil {
			return nil, errors.Wrapf(err, "Error querying BIRD: unable to connect to BIRDv%s socket: %v", ipv, err)
		}
	}
	defer c.Close()

	// To query the current state of the BGP peers, we connect to the BIRD
	// socket and send a "show protocols" message.  BIRD responds with
	// peer data in a table format.
	//
	// Send the request.
	_, err = c.Write([]byte("show protocols\n"))
	if err != nil {
		return nil, errors.Wrapf(err, "Error executing command: unable to write to BIRD socket")
	}

	// Scan the output and collect parsed BGP peers
	log.Debug(context.TODO(), "Reading output from BIRD\n")
	peers, err := scanBIRDPeers(ipv, c)
	if err != nil {
		return nil, errors.Wrapf(err, "Error executing command")
	}

	return peers, nil
}

// scanBIRDPeers scans through BIRD output to return a slice of bgpPeer
// structs.
//
// We split this out from the main printBIRDPeers() function to allow us to
// test this processing in isolation.
func scanBIRDPeers(ipv string, conn net.Conn) ([]bgpPeer, error) {
	// Determine the separator to use for an IP address, based on the
	// IP version.
	ipSep := "."
	if ipv == "6" {
		ipSep = ":"
	}

	// The following is sample output from BIRD
	//
	// 	0001 BIRD 1.5.0 ready.
	// 	2002-name     proto    table    state  since       info
	// 	1002-kernel1  Kernel   master   up     2016-11-21
	//  	 device1  Device   master   up     2016-11-21
	//  	 direct1  Direct   master   up     2016-11-21
	//  	 Mesh_172_17_8_102 BGP      master   up     2016-11-21  Established
	// 	0000
	scanner := bufio.NewScanner(conn)
	peers := []bgpPeer{}

	// Set a time-out for reading from the socket connection.
	err := conn.SetReadDeadline(time.Now().Add(birdTimeOut))
	if err != nil {
		return nil, errors.New("failed to set time-out")
	}

	for scanner.Scan() {
		// Process the next line that has been read by the scanner.
		str := scanner.Text()
		log.Debugf(context.TODO(), "Read: %s\n", str)

		if strings.HasPrefix(str, "0000") { //nolint
			// "0000" means end of data
			break
		} else if strings.HasPrefix(str, "0001") { //nolint
			// "0001" code means BIRD is ready.
		} else if strings.HasPrefix(str, "2002") {
			// "2002" code means start of headings
			f := strings.Fields(str[5:])
			if !reflect.DeepEqual(f, birdExpectedHeadings) {
				return nil, errors.New("unknown BIRD table output format")
			}
		} else if strings.HasPrefix(str, "1002") {
			// "1002" code means first row of data.
			peer := bgpPeer{}
			if peer.unmarshalBIRD(str[5:], ipSep) {
				peers = append(peers, peer)
			}
		} else if strings.HasPrefix(str, " ") {
			// Row starting with a " " is another row of data.
			peer := bgpPeer{}
			if peer.unmarshalBIRD(str[1:], ipSep) {
				peers = append(peers, peer)
			}
		} else {
			// Format of row is unexpected.
			return nil, errors.New("unexpected output line from BIRD")
		}

		// Before reading the next line, adjust the time-out for
		// reading from the socket connection.
		err = conn.SetReadDeadline(time.Now().Add(birdTimeOut))
		if err != nil {
			return nil, errors.New("failed to adjust time-out")
		}
	}

	return peers, scanner.Err()
}

func checkActivePeers(peers []bgpPeer) error {
	activePeers := lo.Reduce(peers, func(agg int, peer bgpPeer, _ int) int {
		// log.Infof(context.TODO(), "+++++++++ %v, %s", peer, peer.State)
		if peer.State == "up" {
			agg++
		}
		return agg
	}, 0)
	if activePeers <= 0 {
		return errors.New("no active peers")
	}
	return nil
}
