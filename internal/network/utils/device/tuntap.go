package device

import (
	"net"

	"github.com/vishvananda/netlink"
)

// Tuntap .
type Tuntap struct {
	*genericLink
}

func createTuntap(attrs netlink.LinkAttrs, d *Driver) *Tuntap {
	attrs.Flags = net.FlagBroadcast | net.FlagMulticast

	var raw = &netlink.Tuntap{LinkAttrs: attrs}
	raw.Mode = netlink.TUNTAP_MODE_TAP
	raw.Flags = netlink.TUNTAP_ONE_QUEUE | netlink.TUNTAP_VNET_HDR
	raw.NonPersist = false

	return newTuntap(raw, d)
}

func newTuntap(raw netlink.Link, d *Driver) *Tuntap {
	return &Tuntap{genericLink: newGenericLink(raw, d)}
}
