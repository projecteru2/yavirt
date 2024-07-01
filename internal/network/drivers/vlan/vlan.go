package vlan

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Handler .
type Handler struct {
	mCol   *MetricsCollector
	subnet int64
}

// New .
func New(subnet int64) *Handler {
	return &Handler{subnet: subnet, mCol: &MetricsCollector{}}
}

func (h *Handler) CheckHealth(_ context.Context) error {
	return nil
}

// NewIP .
func (h *Handler) NewIP(_, _ string) (meta.IP, error) {
	return nil, errors.Wrap(terrors.ErrNotImplemented, "NewIP error")
}

// QueryIPs .
func (h *Handler) QueryIPs(ipns meta.IPNets) ([]meta.IP, error) {
	return h.ipam().Query(context.TODO(), ipns)
}

func (h *Handler) ipam() network.Ipam {
	return NewIpam(h.subnet)
}

// CreateEndpointNetwork .
func (h *Handler) CreateEndpointNetwork(args types.EndpointArgs) (resp types.EndpointArgs, rollback func() error, err error) {
	ip, err := h.ipam().Assign(context.TODO(), &args)
	if err != nil {
		return
	}
	resp = args
	resp.IPs = append(resp.IPs, ip)
	rollback = func() error {
		return h.ipam().Release(context.Background(), ip)
	}
	return
}

// JoinEndpointNetwork .
func (h *Handler) JoinEndpointNetwork(types.EndpointArgs) (rollback func() error, err error) {
	// DO NOTHING
	return
}

// DeleteEndpointNetwork .
func (h *Handler) DeleteEndpointNetwork(args types.EndpointArgs) error {
	return h.ipam().Release(context.Background(), args.IPs...)
}

// QueryIPv4 .
func (h *Handler) QueryIPv4(string) (meta.IP, error) {
	return nil, errors.Wrapf(terrors.ErrNotImplemented, "QueryIPv4 error")
}

// GetCidr .
func (h *Handler) GetCidr() string {
	ip := IP{Value: h.subnet, Subnet: &Subnet{SubnetPrefix: 0}}
	return ip.CIDR()
}

func (h *Handler) CreateNetworkPolicy(string) error {
	return nil
}

func (h *Handler) DeleteNetworkPolicy(string) error {
	return nil
}
