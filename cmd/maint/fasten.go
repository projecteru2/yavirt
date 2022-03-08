package maint

import (
	"context"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/libvirt"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/netx"
	"github.com/projecteru2/yavirt/store"
	"github.com/projecteru2/yavirt/util"
)

var intIPSubnets = map[int64]int64{}

// fasten all local guests that are dangling.
func fasten(c *cli.Context, runtime run.Runtime) error {
	virt, err := libvirt.Connect("qemu:///system")
	if err != nil {
		return errors.Trace(err)
	}
	defer virt.Close()

	ids, err := virt.ListDomainsNames()
	if err != nil {
		return errors.Trace(err)
	}

	prefix := filepath.Join(config.Conf.EtcdPrefix, "ips", "/")
	data, _, err := store.GetPrefix(context.Background(), prefix, (1<<32)-1) //nolint:gomnd // max value of int32
	if err != nil {
		return errors.Trace(err)
	}
	for key := range data {
		if !strings.Contains(key, "occupied") {
			continue
		}
		key = strings.TrimPrefix(key, config.Conf.EtcdPrefix)
		key = strings.TrimLeft(key, "/")
		parts := strings.Split(key, "/")
		intSubnet, err := strconv.ParseInt(parts[1], 10, 64) //nolint
		if err != nil {
			return errors.Annotatef(err, "parse subnet %s failed", parts[1])
		}
		intIP, err := strconv.ParseInt(parts[3], 10, 64) //nolint
		if err != nil {
			return errors.Annotatef(err, "parse subnet %d ip %s failed", intSubnet, parts[3])
		}
		intIPSubnets[intIP] = intSubnet
	}

	for _, id := range ids {
		switch _, err := model.LoadGuest(id); {
		case err == nil:
			fmt.Printf("valid guest: %s\n", id)
			continue

		case errors.IsKeyNotExistsErr(err):
			if err := fastenDangling(id, virt); err != nil {
				return errors.Trace(err)
			}

		default:
			return errors.Trace(err)
		}
	}

	return nil
}

var ips = map[string]string{
	"guest-000104": "10.129.144.1",
	"guest-000160": "10.129.144.24",
	"guest-000172": "10.129.144.28",
	"guest-000145": "10.129.144.15",
	"guest-000189": "10.129.144.39",
	"guest-000128": "10.129.144.11",
	"guest-000156": "10.129.144.21",
	"guest-000175": "10.129.144.30",
	"guest-000157": "10.129.144.22",
	"guest-000174": "10.129.144.32",
	"guest-000155": "10.129.144.20",
	"guest-000173": "10.129.144.31",
	"guest-000144": "10.129.144.16",
	"guest-000188": "10.129.144.38",
	"guest-000164": "10.129.144.26",
	"guest-000184": "10.129.144.36",
	"guest-000150": "10.129.144.18",
	"guest-000152": "10.129.144.19",
	"guest-000183": "10.129.144.34",
	"guest-000168": "10.129.144.27",
	"guest-000177": "10.129.144.35",
	"guest-000140": "10.129.144.14",
	"guest-000186": "10.129.144.37",
	"guest-000158": "10.129.144.23",
	"guest-000171": "10.129.144.29",
	"guest-000114": "10.129.144.4",
	"guest-000162": "10.129.144.25",
	"guest-000180": "10.129.144.33",
	"guest-000138": "10.129.140.16",
	"guest-000190": "10.129.140.36",
	"guest-000170": "10.129.140.28",
	"guest-000130": "10.129.140.13",
	"guest-000154": "10.129.140.20",
	"guest-000181": "10.129.140.32",
	"guest-000167": "10.129.140.26",
	"guest-000169": "10.129.140.27",
	"guest-000166": "10.129.140.25",
	"guest-000179": "10.129.140.31",
	"guest-000117": "10.129.140.7",
	"guest-000187": "10.129.140.35",
	"guest-000165": "10.129.140.24",
	"guest-000182": "10.129.140.33",
	"guest-000163": "10.129.140.23",
	"guest-000195": "10.129.140.37",
	"guest-000129": "10.129.140.12",
	"guest-000159": "10.129.140.21",
	"guest-000185": "10.129.140.34",
	"guest-000151": "10.129.140.18",
	"guest-000176": "10.129.140.30",
	"guest-000141": "10.129.140.17",
	"guest-000153": "10.129.140.19",
	"guest-000178": "10.129.140.29",
	"guest-000142": "10.129.152.13",
	"guest-000107": "10.129.152.2",
	"guest-000136": "10.129.152.6",
	"guest-000192": "10.129.152.15",
	"guest-000143": "10.129.132.10",
	"guest-000194": "10.129.132.13",
	"guest-000137": "10.129.132.8",
	"guest-000191": "10.129.132.11",
	"guest-000139": "10.129.132.9",
	"guest-000193": "10.129.132.12",
}

func fastenDangling(id string, virt *libvirt.Libvirtee) error {
	dom, err := virt.LookupDomain(id)
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	guest, err := model.NewGuest(nil, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if guest.HostName, err = util.Hostname(); err != nil {
		return errors.Trace(err)
	}
	guest.ID = id
	guest.ImageName = "ubuntu1604-sto"

	state, err := dom.GetState()
	if err != nil {
		return errors.Trace(err)
	}
	switch state {
	case libvirt.DomainRunning:
		guest.Status = model.StatusRunning
	case libvirt.DomainShutoff:
		guest.Status = model.StatusStopped
	default:
		return errors.Errorf("doesn't support %s", state)
	}

	info, err := dom.GetInfo()
	if err != nil {
		return errors.Trace(err)
	}
	guest.CPU = int(info.NrVirtCpu)
	guest.Memory = int64(info.MaxMem) * 1024 //nolint

	var flags libvirt.DomainXMLFlags
	txt, err := dom.GetXMLDesc(flags)
	if err != nil {
		return errors.Trace(err)
	}
	dx := domainXML{}
	if err = xml.Unmarshal([]byte(txt), &dx); err != nil {
		return errors.Trace(err)
	}

	for _, disk := range dx.Devices.Disks {
		fn := filepath.Base(disk.Source.File)
		if strings.HasPrefix(fn, "sys-") {
			fn = fn[:len(fn)-4]                               // to remove '.vol' ext.
			id = fn[4:]                                       // to remove 'sys-' prefix.
			if id = strings.TrimLeft(id, "0"); len(id) <= 3 { //nolint
				id = fmt.Sprintf("%06s", id)
			} else {
				id = fmt.Sprintf("%32s", id)
			}
			guest.VolIDs = []string{id}
		}
	}
	if len(guest.VolIDs) < 1 {
		return errors.Errorf("guest %s can't find sys volume", guest.ID)
	}

	ip := ips[guest.ID]
	if len(ip) < 1 {
		fmt.Printf("guest %s hasn't IP, skip it\n", guest.ID)
		return nil
	}
	intIP, err := netx.IPv4ToInt(ip)
	if err != nil {
		return errors.Errorf("guest %s has invalid IP: %s", guest.ID, ip)
	}
	intSubnet, ok := intIPSubnets[intIP]
	if !ok {
		return errors.Errorf("guest %s IP %s hasn't subnet", guest.ID, ip)
	}
	guest.IPNets = meta.IPNets{
		&meta.IPNet{
			IntIP:     intIP,
			IntSubnet: intSubnet,
		},
	}

	res := meta.Resources{guest}
	data, err := res.Encode()
	if err != nil {
		return errors.Trace(err)
	}

	if err := store.Create(context.Background(), data); err != nil {
		return errors.Annotatef(err, "create %s failed", data)
	}

	fmt.Printf("created %s\n", data)

	return nil
}

type domainXML struct {
	Devices struct {
		Disks []struct {
			Source struct {
				File string `xml:"file,attr"`
			} `xml:"source"`
		} `xml:"disk"`
	} `xml:"devices"`
}
