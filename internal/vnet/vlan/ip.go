package vlan

import (
	"context"
	"fmt"
	"net"

	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/device"
)

// IP .
// etcd keys:
//     /ips/<subnet>/free/<ip>
//     /ips/<subnet>/occupied/<ip>
type IP struct {
	Value   int64  `json:"value"`
	GuestID string `json:"guest"`

	*Subnet `json:"-"`

	*meta.Ver `json:"-"`

	occupied bool
}

// NewIP .
func NewIP() *IP {
	return &IP{Ver: meta.NewVer()}
}

// IsAssigned .
func (ip *IP) IsAssigned() bool {
	return ip.occupied
}

// AutoRouteCIDR .
func (ip *IP) AutoRouteCIDR() (string, error) {
	return "", nil
}

// BindGatewayIPNet .
func (ip *IP) BindGatewayIPNet(ipn *net.IPNet) {
	// DO NOTHING
}

// NetIP .
func (ip *IP) NetIP() net.IP {
	return nil
}

// IPNetwork .
func (ip *IP) IPNetwork() *net.IPNet {
	return nil
}

// BindDevice .
func (ip *IP) BindDevice(dev device.VirtLink) {
	// DO NOTHING
}

// BindGuestID .
func (ip *IP) BindGuestID(guestID string) {
	ip.GuestID = guestID
}

// Create .
func (ip *IP) Create() error {
	var ipam = NewIpam("", ip.Subnet.IntSubnet())

	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	return ipam.Insert(ctx, ip)
}

// MetaKey .
func (ip *IP) MetaKey() string {
	if ip.isOccupied() {
		return ip.occupiedKey()
	}
	return ip.freeKey()
}

func (ip *IP) isOccupied() bool {
	return len(ip.GuestID) > 0 || ip.occupied
}

func (ip *IP) freeKey() string {
	return meta.FreeIPKey(ip.Subnet.IntSubnet(), ip.Value)
}

func (ip *IP) occupiedKey() string {
	return meta.OccupiedIPKey(ip.Subnet.IntSubnet(), ip.Value)
}

// CIDR .
func (ip *IP) CIDR() string {
	return fmt.Sprintf("%s/%d", ip.IPAddr(), ip.Prefix())
}

// IPAddr .
func (ip *IP) IPAddr() string {
	return netx.IntToIPv4(ip.IntIP())
}

// IntIP .
func (ip *IP) IntIP() int64 {
	return ip.Value
}

func (ip *IP) String() string {
	var str = fmt.Sprintf("%s, subnet: %s", ip.CIDR(), ip.SubnetAddr())
	if len(ip.GuestID) > 0 {
		str = fmt.Sprintf("%s, bind guest: %s", str, ip.GuestID)
	}
	return str
}

func (ip *IP) equal(b *IP) bool { //nolint
	return ip.Subnet == b.Subnet && ip.Value == b.Value
}

// NetworkMode .
func (ip *IP) NetworkMode() string {
	return vnet.NetworkVlan
}

// NetworkName .
func (ip *IP) NetworkName() (s string) {
	return
}
