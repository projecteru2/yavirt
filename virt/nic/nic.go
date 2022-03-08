package nic

import (
	"bytes"
	"context"
	"fmt"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/virt/agent"
	"github.com/projecteru2/yavirt/virt/types"
)

// Nic .
type Nic struct {
	meta.IP

	ga *agent.Agent
}

// NewNic .
func NewNic(ip meta.IP, ga *agent.Agent) *Nic {
	return &Nic{IP: ip, ga: ga}
}

// Setup .
func (n *Nic) Setup(ctx context.Context, distro, dev string) error {
	if err := n.SaveFile(ctx, distro, dev, dev); err != nil {
		return errors.Trace(err)
	}

	if err := n.addIP(ctx, n.CIDR(), dev); err != nil {
		return errors.Trace(err)
	}

	if err := n.enable(ctx, dev); err != nil {
		return errors.Trace(err)
	}

	if err := n.addRoute(ctx, n.GatewayAddr()); err != nil {
		return errors.Trace(err)
	}

	switch cidr, err := n.AutoRouteCIDR(); {
	case err != nil:
		return errors.Trace(err)

	case len(cidr) > 0:
		if err := n.delRoute(ctx, cidr); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// AddIP .
func (n *Nic) AddIP(ctx context.Context, distro, dev, fn string) error {
	if err := n.SaveFile(ctx, distro, dev, fn); err != nil {
		return errors.Trace(err)
	}

	return n.addIP(ctx, n.CIDR(), dev)
}

// SaveFile .
func (n *Nic) SaveFile(ctx context.Context, distro string, dev, fn string) (err error) {
	var file ConfigFile

	switch distro {
	case model.DistroUbuntu:
		file, err = OpenUbuntuConfigFile(n.ga, dev, fn, n.IP)
	case model.DistroCentOS:
		file, err = OpenCentosConfigFile(n.ga, dev, n.IP)
	default:
		err = errors.Annotatef(errors.ErrInvalidValue, "invalid distro: %s", distro)
	}

	if err != nil {
		return errors.Trace(err)
	}

	defer file.Close()

	return file.Save()
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
