package ovn

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// IP .
// etcd keys:
//
//	/ips/<subnet>/free/<ip>
//	/ips/<subnet>/occupied/<ip>
type IP struct {
	IPNet        *net.IPNet
	IP           net.IP
	Device       device.VirtLink
	GatewayIPNet *net.IPNet
	GuestID      string `json:"guest"`
	*meta.Ver           // just for fulfilling meta.IP, it doesn't be referred.
}

// NewIP .
func NewIP(ipStr, subnet string) (*IP, error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}
	if ipStr != "" {
		ip = net.ParseIP(ipStr)
	}
	gwIP, err := netx.DefaultGatewayIP(ipnet)
	if err != nil {
		return nil, err
	}
	gwIPNet := &net.IPNet{
		IP:   gwIP,
		Mask: ipnet.Mask,
	}
	ans := &IP{
		IPNet:        ipnet,
		IP:           ip,
		GatewayIPNet: gwIPNet,
	}
	return ans, nil
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
	var prefix, _ = ip.IPNet.Mask.Size()
	return prefix
}

func (ip *IP) String() string {
	return ip.CIDR()
}

// AutoRouteCIDR .
func (ip *IP) AutoRouteCIDR() (string, error) {
	var _, ipn, err = netx.ParseCIDR(ip.CIDR())
	if err != nil {
		return "", errors.Wrap(err, "")
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
	return ip.IPNet
}

// Netmask .
func (ip *IP) Netmask() string {
	mask := ip.IPNet.Mask
	var s = make([]string, len(mask))

	for i, byte := range mask {
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
	var prefix, _ = ip.IPNet.Mask.Size()
	return prefix
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
	return network.OVNMode
}

// NetworkName .
func (ip *IP) NetworkName() (s string) {
	return
}
