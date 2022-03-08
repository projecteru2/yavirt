package calico

import (
	"fmt"
	"net"
	"strings"

	calinet "github.com/projectcalico/libcalico-go/lib/net"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/device"
)

// IP .
type IP struct {
	*calinet.IPNet

	GatewayIPNet *net.IPNet

	Device device.VirtLink

	GuestID string

	*meta.Ver // just for fulfilling meta.IP, it doesn't be referred.
}

// ParseCIDR .
func ParseCIDR(cidr string) (*IP, error) {
	var _, ipn, err = parseCIDR(cidr)
	if err != nil {
		return nil, errors.Annotatef(err, cidr)
	}
	return NewIP(ipn), nil
}

func parseCIDR(cidr string) (ip *calinet.IP, ipn *calinet.IPNet, err error) {
	if ip, ipn, err = calinet.ParseCIDR(cidr); err != nil {
		return
	}

	ipn.IP = ip.IP

	return
}

// NewIP .
func NewIP(ipn *calinet.IPNet) *IP {
	return &IP{IPNet: ipn}
}

// BindGatewayIPNet .
func (ip *IP) BindGatewayIPNet(ipn *net.IPNet) {
	ip.GatewayIPNet = ipn
}

// BindDevice .
func (ip *IP) BindDevice(dev device.VirtLink) {
	ip.Device = dev
}

// BindGuestID .
func (ip *IP) BindGuestID(guestID string) {
	ip.GuestID = guestID
}

// IntIP .
func (ip *IP) IntIP() (v int64) {
	v, _ = netx.IPv4ToInt(ip.IP.String())
	return
}

// IntSubnet .
func (ip *IP) IntSubnet() int64 {
	return 0
}

// Prefix .
func (ip *IP) Prefix() int {
	var prefix, _ = ip.Mask.Size()
	return prefix
}

func (ip *IP) String() string {
	return ip.CIDR()
}

// AutoRouteCIDR .
func (ip *IP) AutoRouteCIDR() (string, error) {
	var _, ipn, err = netx.ParseCIDR(ip.CIDR())
	if err != nil {
		return "", errors.Trace(err)
	}
	return ipn.String(), nil
}

// CIDR .
func (ip *IP) CIDR() string {
	return ip.IPNet.String()
}

// NetIP .
func (ip *IP) NetIP() net.IP {
	return ip.IP
}

// IPNetwork .
func (ip *IP) IPNetwork() *net.IPNet {
	return &ip.IPNet.IPNet
}

// Netmask .
func (ip *IP) Netmask() string {
	var s = make([]string, len(ip.Mask))

	for i, byte := range ip.Mask {
		s[i] = fmt.Sprintf("%d", byte)
	}

	return strings.Join(s, ".")
}

// IPAddr .
func (ip *IP) IPAddr() string {
	return ip.IP.String()
}

// SubnetAddr .
func (ip *IP) SubnetAddr() string {
	return ""
}

// IntGateway .
func (ip *IP) IntGateway() (v int64) {
	v, _ = netx.IPv4ToInt(ip.GatewayAddr())
	return v
}

// GatewayPrefix .
func (ip *IP) GatewayPrefix() int {
	return net.IPv4len * 8 //nolint:gomnd // Always /32
}

// GatewayAddr .
func (ip *IP) GatewayAddr() string {
	return ip.GatewayIPNet.IP.String()
}

// IsAssigned always returns true,
// the unassigned IPs're only stored in Calico itself.
func (ip *IP) IsAssigned() bool {
	return true
}

// MetaKey .
func (ip *IP) MetaKey() string {
	// DOES NOT STORE
	return ""
}

// NetworkMode .
func (ip *IP) NetworkMode() string {
	return vnet.NetworkCalico
}

// NetworkName .
func (ip *IP) NetworkName() (s string) {
	return
}
