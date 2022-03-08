package vlan

import (
	"fmt"

	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// Subnet .
type Subnet struct {
	Subnet       int64 `json:"subnet"`
	SubnetPrefix int   `json:"prefix"`
	Gateway      int64 `json:"gateway"`

	*meta.Ver
}

// LoadSubnet .
func LoadSubnet(sub int64) (*Subnet, error) {
	var subnet = NewSubnet(sub)
	return subnet, meta.Load(subnet)
}

// NewSubnet .
func NewSubnet(sub int64) *Subnet {
	return &Subnet{Subnet: sub, Ver: meta.NewVer()}
}

// Create .
func (s *Subnet) Create() error {
	return meta.Create(meta.Resources{s})
}

// MetaKey .
func (s *Subnet) MetaKey() string {
	return meta.SubnetKey(s.IntSubnet())
}

func (s *Subnet) String() string {
	return fmt.Sprintf("%s/%d, gateway: %s", s.SubnetAddr(), s.Prefix(), s.GatewayAddr())
}

// IntGateway .
func (s *Subnet) IntGateway() int64 {
	return s.Gateway
}

// GatewayAddr .
func (s *Subnet) GatewayAddr() string {
	return netx.IntToIPv4(s.Gateway)
}

// GatewayPrefix .
func (s *Subnet) GatewayPrefix() int {
	return 32 //nolint
}

// SubnetAddr .
func (s *Subnet) SubnetAddr() string {
	return netx.IntToIPv4(s.IntSubnet())
}

// IntSubnet .
func (s *Subnet) IntSubnet() int64 {
	return s.Subnet
}

// Netmask .
func (s *Subnet) Netmask() string {
	return netx.PrefixToNetmask(s.Prefix())
}

// Prefix .
func (s *Subnet) Prefix() int {
	return s.SubnetPrefix
}
