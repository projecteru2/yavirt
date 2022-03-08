package device

import (
	"strings"

	"github.com/vishvananda/netlink"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Addr .
type Addr struct {
	*netlink.Addr
}

func newAddr(raw *netlink.Addr) *Addr {
	return &Addr{Addr: raw}
}

func (a *Addr) String() string {
	return a.IPNet.String()
}

// Addrs .
type Addrs []*Addr

func newAddrs(raw []netlink.Addr) Addrs {
	var addrs = make(Addrs, len(raw))
	for i := range raw {
		addrs[i] = newAddr(&raw[i])
	}
	return addrs
}

func (a Addrs) String() string {
	var strs = make([]string, a.Len())
	for i, x := range a {
		strs[i] = x.String()
	}
	return strings.Join(strs, ", ")
}

// Len .
func (a Addrs) Len() int {
	return len(a)
}

// BindAddr .
func (d *Driver) BindAddr(cidr, linkName string) error {
	addr, err := d.ParseCIDR(cidr)
	if err != nil {
		return errors.Trace(err)
	}

	link, err := d.ShowLink(linkName)
	if err != nil {
		return errors.Trace(err)
	}

	return link.BindAddr(addr)
}

// ParseCIDR .
func (d *Driver) ParseCIDR(cidr string) (*Addr, error) {
	var raw, err = netlink.ParseAddr(cidr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &Addr{Addr: raw}, nil
}

// ListAddrs .
func (d *Driver) ListAddrs(linkName string, family int) (Addrs, error) {
	link, err := d.ShowLink(linkName)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return link.ListAddr()
}
