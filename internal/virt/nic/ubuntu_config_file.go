package nic

import (
	"fmt"

	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
)

// UbuntuConfigFile .
type UbuntuConfigFile struct {
	*GenericConfigFile
}

// OpenUbuntuConfigFile .
func OpenUbuntuConfigFile(ga *agent.Agent, dev, fn string, ip meta.IP) (fw *UbuntuConfigFile, err error) {
	fw = &UbuntuConfigFile{GenericConfigFile: &GenericConfigFile{
		path: getEthUbuntuFile(fn),
		ip:   ip,
		dev:  dev,
	}}
	fw.file, err = agent.OpenFile(ga, fw.path, "w")
	return
}

// Save .
func (w *UbuntuConfigFile) Save() (err error) {
	var conf = fmt.Sprintf(`auto %s
iface %s inet static
    address %s
    netmask %s
`, w.dev, w.dev, w.ip.IPAddr(), w.ip.Netmask())

	if gw := w.ip.GatewayAddr(); len(gw) > 0 {
		conf += fmt.Sprintf("    gateway %s\n", gw)
	}

	return w.Write([]byte(conf))
}
