package device

import (
	"fmt"
	"net"

	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/vishvananda/netlink"

	"github.com/cockroachdb/errors"
)

type genericLink struct {
	netlink.Link

	*Driver
}

func newGenericLink(raw netlink.Link, d *Driver) *genericLink {
	return &genericLink{
		Link:   raw,
		Driver: d,
	}
}

func (g *genericLink) BindAddr(addr *Addr) error {
	g.Lock()
	defer g.Unlock()

	var err = g.AddrAdd(g, addr.Addr)
	if err != nil && isFileExistsErr(err) {
		return errors.Wrapf(terrors.ErrVirtLinkAddrExists, "%s", addr)
	}

	return err
}

func (g *genericLink) ListAddr() (Addrs, error) {
	var raw, err = g.AddrList(g, FamilyIPv4)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return newAddrs(raw), nil
}

func (g *genericLink) Add() error {
	g.Lock()
	defer g.Unlock()

	if _, err := g.showLink(g.Name()); err == nil {
		return errors.Wrapf(terrors.ErrVirtLinkExists, g.Name())
	}

	return g.LinkAdd(g.Link)
}

func (g *genericLink) Down() error {
	g.Lock()
	defer g.Unlock()
	return g.LinkSetDown(g)
}

func (g *genericLink) Up() error {
	g.Lock()
	defer g.Unlock()
	return g.LinkSetUp(g)
}

func (g *genericLink) AddRoute(dest, src string) error {
	g.Lock()
	defer g.Unlock()
	return g.addRoute(dest, src, g)
}

func (g *genericLink) ClearRoutes() error {
	g.Lock()
	defer g.Unlock()

	var raw, err = g.RouteList(g, FamilyIPv4)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, r := range raw {
		if err := g.deleteRoute(r.Dst); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func (g *genericLink) DeleteRoute(cidr string) error {
	g.Lock()
	defer g.Unlock()

	var raw, err = g.RouteList(g, FamilyIPv4)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, r := range raw {
		if r.Dst.String() != cidr {
			continue
		}

		return g.deleteRoute(r.Dst)
	}

	// There's no available route rule for the destination.
	return nil
}

func (g *genericLink) String() string {
	return fmt.Sprintf("%s: %s: %v, %s", g.Name(), g.Type(), g.Flags(), g.addrsString())
}

func (g *genericLink) MAC() string {
	return g.Attrs().HardwareAddr.String()
}

func (g *genericLink) Name() string {
	return g.Attrs().Name
}

func (g *genericLink) Index() int {
	return g.Attrs().Index
}

func (g *genericLink) Flags() net.Flags {
	return g.Attrs().Flags
}

func (g *genericLink) addrsString() string {
	switch addrs, err := g.ListAddr(); {
	case err != nil:
		return err.Error()
	case addrs.Len() > 0:
		return addrs.String()
	default:
		return "<NOIP>"
	}
}
