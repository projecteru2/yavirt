package vlan

import (
	"context"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/internal/vnet/ipam"
	"github.com/projecteru2/yavirt/internal/vnet/types"
	vlannet "github.com/projecteru2/yavirt/internal/vnet/vlan"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// Handler .
type Handler struct {
	guestID string
	subnet  int64
}

// New .
func New(guestID string, subnet int64) *Handler {
	return &Handler{guestID: guestID, subnet: subnet}
}

// NewIP .
func (h *Handler) NewIP(_, _ string) (meta.IP, error) {
	return nil, errors.Trace(errors.ErrNotImplemented)
}

// AssignIP .
func (h *Handler) AssignIP() (meta.IP, error) {
	return h.ipam().Assign(context.Background())
}

// ReleaseIPs .
func (h *Handler) ReleaseIPs(ips ...meta.IP) error {
	return h.ipam().Release(context.Background(), ips...)
}

// QueryIPs .
func (h *Handler) QueryIPs(ipns meta.IPNets) ([]meta.IP, error) {
	return h.ipam().Query(context.Background(), ipns)
}

func (h *Handler) ipam() ipam.Ipam {
	return vlannet.NewIpam(h.guestID, h.subnet)
}

// CreateEndpointNetwork .
func (h *Handler) CreateEndpointNetwork(types.EndpointArgs) (resp types.EndpointArgs, rollback func(), err error) {
	return
}

// JoinEndpointNetwork .
func (h *Handler) JoinEndpointNetwork(types.EndpointArgs) (rollback func(), err error) {
	// DO NOTHING
	return
}

// DeleteEndpointNetwork .
func (h *Handler) DeleteEndpointNetwork(types.EndpointArgs) error {
	// DO NOTHING
	return nil
}

// GetEndpointDevice .
func (h *Handler) GetEndpointDevice(string) (device.VirtLink, error) {
	// DO NOTHING
	return nil, nil
}

// QueryIPv4 .
func (h *Handler) QueryIPv4(string) (meta.IP, error) {
	return nil, errors.Trace(errors.ErrNotImplemented)
}

// GetCidr .
func (h *Handler) GetCidr() string {
	ip := vlannet.IP{Value: h.subnet, Subnet: &vlannet.Subnet{SubnetPrefix: 0}}
	return ip.CIDR()
}
