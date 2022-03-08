package meta

import (
	"fmt"
	"net"

	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// IP .
type IP interface {
	Resource

	BindDevice(device.VirtLink)
	BindGuestID(guestID string)
	BindGatewayIPNet(*net.IPNet)
	IsAssigned() bool

	IntIP() int64
	IntSubnet() int64
	IntGateway() int64

	Prefix() int
	GatewayPrefix() int

	CIDR() string
	AutoRouteCIDR() (string, error)
	Netmask() string
	IPAddr() string
	NetIP() net.IP
	IPNetwork() *net.IPNet
	SubnetAddr() string
	GatewayAddr() string

	NetworkMode() string
	NetworkName() string
}

// IPNets .
type IPNets []*IPNet

// IPNet .
type IPNet struct {
	Network       string `json:"network,omitempty"`
	IntSubnet     int64  `json:"subnet,omitempty"`
	IntIP         int64  `json:"ip,omitempty"`
	IntGateway    int64  `json:"gateway,omitempty"`
	IPPrefix      int    `json:"ip_prefix,omitempty"`
	GatewayPrefix int    `json:"gatway_prefix,omitempty"`
	Assigned      bool   `json:"-"`
}

// IPv4 .
func (ipn IPNet) IPv4() string {
	return netx.IntToIPv4(ipn.IntIP)
}

// CIDR .
func (ipn IPNet) CIDR() string {
	return fmt.Sprintf("%s/%d", netx.IntToIPv4(ipn.IntIP), ipn.IPPrefix)
}

// GatewayIPNet .
func (ipn IPNet) GatewayIPNet() (*net.IPNet, error) {
	return netx.ParseCIDR2(ipn.GatewayCIDR())
}

// GatewayCIDR .
func (ipn IPNet) GatewayCIDR() string {
	return fmt.Sprintf("%s/%d", netx.IntToIPv4(ipn.IntGateway), ipn.GatewayPrefix)
}

// ConvIPNets .
func ConvIPNets(ips []IP) IPNets {
	var ipns = make(IPNets, len(ips))
	for i, ip := range ips {
		ipns[i] = ParseIPNet(ip)
	}
	return ipns
}

// ParseIPNet .
func ParseIPNet(ip IP) *IPNet {
	return &IPNet{
		IntIP:         ip.IntIP(),
		IntSubnet:     ip.IntSubnet(),
		IntGateway:    ip.IntGateway(),
		IPPrefix:      ip.Prefix(),
		GatewayPrefix: ip.GatewayPrefix(),
	}
}
