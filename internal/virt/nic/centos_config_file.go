package nic

import (
	"fmt"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
)

// CentosConfigFile .
type CentosConfigFile struct {
	*GenericConfigFile
}

// OpenCentosConfigFile .
func OpenCentosConfigFile(ga *agent.Agent, dev string, ip meta.IP) (fw *CentosConfigFile, err error) {
	fw = &CentosConfigFile{GenericConfigFile: &GenericConfigFile{
		path: getEthCentOSFile(dev),
		ip:   ip,
		dev:  dev,
	}}
	fw.file, err = agent.OpenFile(ga, fw.path, "w")
	return
}

// Save .
func (w *CentosConfigFile) Save() (err error) {
	var conf = fmt.Sprintf(`TYPE="Ethernet"
BOOTPROTO="static"
NAME="%s"
DEVICE="%s"
ONBOOT="yes"
IPADDR="%s"
PREFIX="%d"
GATEWAY="%s"
`, w.dev, w.dev, w.ip.IPAddr(), w.ip.Prefix(), w.ip.GatewayAddr())

	return w.Write([]byte(conf))
}
