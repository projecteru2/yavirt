package fake

import (
	"context"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/prometheus/client_golang/prometheus"
)

type Driver struct{}

func (d *Driver) CheckHealth(_ context.Context) error {
	return nil
}
func (d *Driver) GetMetricsCollector() prometheus.Collector {
	return nil
}

func (d *Driver) QueryIPs(meta.IPNets) ([]meta.IP, error) {
	return nil, nil
}
func (d *Driver) CreateEndpointNetwork(types.EndpointArgs) (types.EndpointArgs, func() error, error) {
	return types.EndpointArgs{}, nil, nil
}
func (d *Driver) JoinEndpointNetwork(types.EndpointArgs) (func() error, error) {
	return nil, nil //nolint:nilnil
}
func (d *Driver) DeleteEndpointNetwork(types.EndpointArgs) error {
	return nil
}
func (d *Driver) GetEndpointDevice(string) (device.VirtLink, error) {
	return nil, nil //nolint
}
func (d *Driver) CreateNetworkPolicy(string) error {
	return nil
}
func (d *Driver) DeleteNetworkPolicy(string) error {
	return nil
}
