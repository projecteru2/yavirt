package device

import (
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	// MTU .
	MTU = 1500
	// Qlen .
	Qlen = 1000
	// FamilyIPv4 .
	FamilyIPv4 = netlink.FAMILY_V4
	// RouteTableMain .
	RouteTableMain = unix.RT_TABLE_MAIN
	// RouteScopeLink .
	RouteScopeLink = netlink.SCOPE_LINK
	// LinkTypeDummy .
	LinkTypeDummy = "dummy"
	// LinkTypeTuntap .
	LinkTypeTuntap = "tuntap"
	// LinkTypeTun .
	LinkTypeTun = "tun"
)
