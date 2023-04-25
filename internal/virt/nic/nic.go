package nic

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"path"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

//go:embed templates/vm-init.sh
var vm_init_script string

type NicList struct {
	ips []meta.IP
	ga  *agent.Agent
}

// NewNic .
func NewNicList(ips []meta.IP, ga *agent.Agent) *NicList {
	return &NicList{ips: ips, ga: ga}
}

// Setup .
func (nl *NicList) Setup(ctx context.Context) error {
	var args []string
	for _, ip := range nl.ips {
		args = append(args, ip.CIDR(), ip.GatewayAddr())
	}
	log.Infof("Setup NIc list %v\n", nl.ips)
	if err := nl.execVmInitScript(ctx, args...); err != nil {
		return errors.Trace(err)
	}
	for idx, ip := range nl.ips {
		switch cidr, err := ip.AutoRouteCIDR(); {
		case err != nil:
			return errors.Trace(err)

		case len(cidr) > 0:
			n, err := nl.GetNic(idx)
			if err != nil {
				return err
			}
			if err := n.delRoute(ctx, cidr); err != nil {
				return errors.Trace(err)
			}
		}
	}
	return nil
}

func (nl *NicList) GetNic(idx int) (*Nic, error) {
	if idx >= len(nl.ips) {
		return nil, fmt.Errorf("NicList has only %v ips, so can't get Nic with index %v", len(nl.ips), idx)
	}
	return &Nic{
		IP: nl.ips[idx],
		ga: nl.ga,
	}, nil
}

func (n *NicList) execVmInitScript(ctx context.Context, args ...string) error {
	vm_fname := "/tmp/vm-init.sh"
	if err := writeFileToGuest(ctx, n.ga, []byte(vm_init_script), vm_fname); err != nil {
		return errors.Trace(err)
	}
	new_args := []string{vm_fname}
	new_args = append(new_args, args...)

	var st = <-n.ga.Exec(ctx, "bash", new_args...)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, fmt.Sprintf("failed to run vm-init.sh %v", args))
	}

	st = <-n.ga.Exec(ctx, "systemctl", "restart", "systemd-networkd")
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, fmt.Sprintf("failed to restart networkd"))
	}

	st = <-n.ga.Exec(ctx, "rm", "-f", vm_fname)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, fmt.Sprintf("failed to remove vm-init.sh"))
	}

	return nil
}

// Nic .
type Nic struct {
	meta.IP

	ga *agent.Agent
}

// NewNic .
func NewNic(ip meta.IP, ga *agent.Agent) *Nic {
	return &Nic{IP: ip, ga: ga}
}

// AddIP .
func (n *Nic) AddIP(ctx context.Context, distro, dev, fn string) error {
	if err := n.persisteNetworkCfg(ctx, dev); err != nil {
		return errors.Trace(err)
	}

	return n.addIP(ctx, n.CIDR(), dev)
}

func writeFileToGuest(ctx context.Context, ga *agent.Agent, buf []byte, fname string) (err error) {
	var fp agent.File
	fp, err = agent.OpenFile(ga, fname, "w")
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = fp.Flush()
		}
	}()

	_, err = fp.Write(buf)

	return
}

func (n *Nic) persisteNetworkCfg(ctx context.Context, dev string) (err error) {
	// TODO: set gateway only when gateway is available
	var netCfg = fmt.Sprintf(`
[Match]
Name=%s

[Network]
Address=%s
Gateway=%s
`, dev, n.CIDR(), n.GatewayAddr())

	{
		fname := fmt.Sprintf("10-%s.network", dev)
		p := path.Join("/etc/systemd/network", fname)
		if err = writeFileToGuest(ctx, n.ga, []byte(netCfg), p); err != nil {
			return err
		}
	}
	// restart networkd
	var st = <-n.ga.Exec(ctx, "systemctl", "restart", "systemd-networkd")
	if err = st.Error(); err != nil {
		return errors.Trace(err)
	}
	return
	// if gw := w.ip.GatewayAddr(); len(gw) > 0 {
	// 	conf += fmt.Sprintf("    gateway %s\n", gw)
	// }

	// return w.Write([]byte(conf))
}

func (n *Nic) enable(ctx context.Context, dev string) error {
	var st = <-n.ga.Exec(ctx, "ip", "link", "set", dev, "up")
	if err := st.Error(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (n *Nic) addRoute(ctx context.Context, dest string) error {
	return n.doIP(ctx, "ip", "route", "add", "default", "via", dest)
}

func (n *Nic) delRoute(ctx context.Context, dest string) error {
	return n.doIP(ctx, "ip", "route", "del", dest)
}

func (n *Nic) addIP(ctx context.Context, cidr, dev string) error {
	return n.doIP(ctx, "ip", "a", "add", cidr, "dev", dev)
}

func (n *Nic) doIP(ctx context.Context, cmd string, args ...string) error {
	var st = <-n.ga.ExecOutput(ctx, cmd, args...)
	_, _, err := st.CheckStdio(func(_, se []byte) bool {
		return bytes.HasSuffix(bytes.Trim(se, "\n"), []byte(" File exists"))
	})
	return errors.Annotatef(err, fmt.Sprintf("%s %v failed", cmd, args))
}

// GetEthFile .
func GetEthFile(distro, dev string) (string, error) {
	switch distro {
	case types.Ubuntu:
		return getEthUbuntuFile(dev), nil
	case types.CentOS:
		return getEthCentOSFile(dev), nil
	default:
		return "", errors.Annotatef(errors.ErrInvalidValue, "invalid distro: %s", distro)
	}
}

func getEthUbuntuFile(dev string) string {
	return fmt.Sprintf(types.EthUbuntuFileFmt, dev)
}

func getEthCentOSFile(dev string) string {
	return fmt.Sprintf(types.EthCentOSFileFmt, dev)
}
