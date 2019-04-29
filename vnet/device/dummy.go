package device

import (
	"net"

	"github.com/vishvananda/netlink"
)

// Dummy .
type Dummy struct {
	*genericLink
}

func createDummy(attrs netlink.LinkAttrs, d *Driver) *Dummy {
	attrs.Flags = net.FlagBroadcast

	var raw = &netlink.Dummy{LinkAttrs: attrs}

	return newDummy(raw, d)
}

func newDummy(raw netlink.Link, d *Driver) *Dummy {
	return &Dummy{genericLink: newGenericLink(raw, d)}
}

// NilDummy .
func NilDummy() *Dummy {
	var raw = &netlink.Dummy{LinkAttrs: netlink.NewLinkAttrs()}
	return &Dummy{genericLink: newGenericLink(raw, nil)}
}
