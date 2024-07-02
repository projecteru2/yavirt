package boar

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/network"
	networkFactory "github.com/projecteru2/yavirt/internal/network/factory"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/guest"

	calihandler "github.com/projecteru2/yavirt/internal/network/drivers/calico"
	vlanhandler "github.com/projecteru2/yavirt/internal/network/drivers/vlan"
)

// ConnectNetwork .
func (svc *Boar) ConnectNetwork(ctx context.Context, id, network, ipv4 string) (cidr string, err error) {
	var ip meta.IP

	if err := svc.ctrl(ctx, id, intertypes.MiscOp, func(g *guest.Guest) (ce error) {
		ip, ce = g.ConnectExtraNetwork(network, ipv4)
		return ce
	}, nil); err != nil {
		log.WithFunc("boar.ConnectNetwork").Error(ctx, err)
		metrics.IncrError()
		return "", errors.Wrap(err, "")
	}

	return ip.CIDR(), nil
}

// DisconnectNetwork .
func (svc *Boar) DisconnectNetwork(ctx context.Context, id, network string) (err error) {
	err = svc.ctrl(ctx, id, intertypes.MiscOp, func(g *guest.Guest) error {
		return g.DisconnectExtraNetwork(network)
	}, nil)
	if err != nil {
		log.WithFunc("DisconnectNetwork").Error(ctx, err)
		metrics.IncrError()
	}
	return
}

// NetworkList .
func (svc *Boar) NetworkList(ctx context.Context, drivers []string) ([]*types.Network, error) {
	drv := map[string]struct{}{}
	for _, driver := range drivers {
		drv[driver] = struct{}{}
	}

	networks := []*types.Network{}
	for mode, hand := range networkFactory.ListDrivers() {
		switch mode {
		case network.CalicoMode:
			if _, ok := drv[network.CalicoMode]; !ok {
				break
			}
			caliHandler, ok := hand.(*calihandler.Driver)
			if !ok {
				break
			}
			for _, poolName := range caliHandler.PoolNames() {
				subnet, err := caliHandler.GetIPPoolCidr(ctx, poolName)
				if err != nil {
					log.WithFunc("NetworkList").Error(ctx, err)
					metrics.IncrError()
					return nil, err
				}

				networks = append(networks, &types.Network{
					Name:    poolName,
					Subnets: []string{subnet},
				})
			}
			return networks, nil
		case network.VlanMode: // vlan
			if _, ok := drv[network.VlanMode]; !ok {
				break
			}
			handler := vlanhandler.New(svc.Host.Subnet)
			networks = append(networks, &types.Network{
				Name:    "vlan",
				Subnets: []string{handler.GetCidr()},
			})
		}
	}
	return networks, nil
}
