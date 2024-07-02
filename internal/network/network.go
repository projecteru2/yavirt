package network

import (
	"context"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/prometheus/client_golang/prometheus"
)

type Driver interface {
	// Prepare(ctx context.Context) error

	CheckHealth(ctx context.Context) error
	QueryIPs(meta.IPNets) ([]meta.IP, error)

	CreateEndpointNetwork(types.EndpointArgs) (types.EndpointArgs, func() error, error)
	JoinEndpointNetwork(types.EndpointArgs) (func() error, error)
	DeleteEndpointNetwork(types.EndpointArgs) error

	CreateNetworkPolicy(string) error
	DeleteNetworkPolicy(string) error

	GetMetricsCollector() prometheus.Collector
}

// Ipam .
type Ipam interface {
	Assign(ctx context.Context, args *types.EndpointArgs) (meta.IP, error)
	Release(context.Context, ...meta.IP) error
	Query(context.Context, meta.IPNets) ([]meta.IP, error)
}
