package nic

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"path"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

//go:embed templates/vm-init.sh
var vmInitScript string

type NICs struct {
	ips []meta.IP
	ga  *agent.Agent
}

// NewNic .
func NewNicList(ips []meta.IP, ga *agent.Agent) *NICs {
	return &NICs{ips: ips, ga: ga}
}

// Setup .
func (nl *NICs) Setup(ctx context.Context) error {
	args := make([]string, 0, 2*len(nl.ips))
	for _, ip := range nl.ips {
		args = append(args, ip.CIDR(), ip.GatewayAddr())
	}
	log.Infof(ctx, "Setup NIC list %v", nl.ips)
	if err := nl.execVMInitScript(ctx, args...); err != nil {
		return errors.Wrap(err, "")
	}
	for idx, ip := range nl.ips {
		switch cidr, err := ip.AutoRouteCIDR(); {
		case err != nil:
			return errors.Wrap(err, "")

		case len(cidr) > 0:
			n, err := nl.GetNic(idx)
			if err != nil {
				return err
			}
			if err := n.delRoute(ctx, cidr); err != nil {
				return errors.Wrap(err, "")
			}
		}
	}
	return nil
}

func (nl *NICs) GetNic(idx int) (*Nic, error) {
	if idx >= len(nl.ips) {
		return nil, fmt.Errorf("NicList has only %v ips, so can't get Nic with index %v", len(nl.ips), idx)
	}
	return &Nic{
		IP: nl.ips[idx],
		ga: nl.ga,
	}, nil
}

func (nl *NICs) execVMInitScript(ctx context.Context, args ...string) error {
	vmFname := "/tmp/vm-init.sh"
	if err := writeFileToGuest(ctx, nl.ga, []byte(vmInitScript), vmFname); err != nil {
		return errors.Wrap(err, "")
	}
	newArgs := []string{vmFname}
	newArgs = append(newArgs, args...)

	var st = <-nl.ga.Exec(ctx, "bash", newArgs...)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "failed to run vm-init.sh %v", args)
	}

	st = <-nl.ga.Exec(ctx, "systemctl", "restart", "systemd-networkd")
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "failed to restart networkd")
	}

	st = <-nl.ga.Exec(ctx, "rm", "-f", vmFname)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "failed to remove vm-init.sh")
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
func (n *Nic) AddIP(ctx context.Context, dev string) error {
	if err := n.persisteNetworkCfg(ctx, dev); err != nil {
		return errors.Wrap(err, "")
	}

	return n.addIP(ctx, n.CIDR(), dev)
}

func writeFileToGuest(ctx context.Context, ga *agent.Agent, buf []byte, fname string) (err error) {
	var fp agent.File
	fp, err = agent.OpenFile(ctx, ga, fname, "w")
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = fp.Flush(ctx)
		}
	}()

	_, err = fp.Write(ctx, buf)

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
		return errors.Wrap(err, "")
	}
	return
	// if gw := w.ip.GatewayAddr(); len(gw) > 0 {
	// 	conf += fmt.Sprintf("    gateway %s\n", gw)
	// }

	// return w.Write([]byte(conf))
}

func (n *Nic) enable(ctx context.Context, dev string) error { //nolint
	var st = <-n.ga.Exec(ctx, "ip", "link", "set", dev, "up")
	if err := st.Error(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (n *Nic) addRoute(ctx context.Context, dest string) error { //nolint
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
	return errors.Wrapf(err, "%s %v failed", cmd, args)
}

// GetEthFile .
func GetEthFile(distro, dev string) (string, error) {
	switch distro {
	case types.Ubuntu:
		return getEthUbuntuFile(dev), nil
	case types.CentOS:
		return getEthCentOSFile(dev), nil
	default:
		return "", errors.Wrapf(terrors.ErrInvalidValue, "invalid distro: %s", distro)
	}
}

func getEthUbuntuFile(dev string) string {
	return fmt.Sprintf(types.EthUbuntuFileFmt, dev)
}

func getEthCentOSFile(dev string) string {
	return fmt.Sprintf(types.EthCentOSFileFmt, dev)
}
