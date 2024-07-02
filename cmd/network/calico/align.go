package calico

import (
	"context"
	"fmt"
	"net"

	libcalinet "github.com/projectcalico/calico/libcalico-go/lib/net"
	libcaliopt "github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/network/drivers/calico"
	networkFactory "github.com/projecteru2/yavirt/internal/network/factory"
)

func alignFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "dry-run",
			Value: false,
		},
	}
}

func align(c *cli.Context, _ run.Runtime) error {
	bound, err := getGatewayBoundIPs()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return alignGatewayIPs(bound, c.Bool("dry-run"))
}

func getGatewayBoundIPs() ([]net.IP, error) {
	drv := networkFactory.GetDriver(network.CalicoMode)
	if drv == nil {
		return nil, errors.New("calico driver is not intialized")
	}
	cali, _ := drv.(*calico.Driver)
	if err := cali.InitGateway("yavirt-cali-gw"); err != nil {
		return nil, errors.Wrap(err, "")
	}

	gw := cali.Gateway()
	addrs, err := gw.ListAddr()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ips := make([]net.IP, addrs.Len())
	for i, addr := range addrs {
		ips[i] = addr.IPNet.IP
	}

	return ips, nil
}

func alignGatewayIPs(bound []net.IP, dryRun bool) error {
	drv := networkFactory.GetDriver(network.CalicoMode)
	if drv == nil {
		return errors.New("calico driver is not intialized")
	}
	cali, _ := drv.(*calico.Driver)
	wep := cali.GatewayWorkloadEndpoint()

	for _, bip := range bound {
		ipn := libcalinet.IPNet{
			IPNet: net.IPNet{
				IP:   bip,
				Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8),
			},
		}

		var found bool
		for _, existsCIDR := range wep.Spec.IPNetworks {
			if existsCIDR == ipn.String() {
				found = true
				break
			}
		}

		if found {
			continue
		}

		wep.Spec.IPNetworks = append(wep.Spec.IPNetworks, ipn.String())

		fmt.Printf("%s doesn't belong wep %s\n", ipn, wep.Name)
		fmt.Printf("up %s to %v\n\n", wep.Name, wep.Spec.IPNetworks)

		if dryRun {
			continue
		}

		drv := networkFactory.GetDriver(network.CalicoMode)
		if drv == nil {
			return errors.New("calico driver is not intialized")
		}
		cali, _ := drv.(*calico.Driver)
		_, err := cali.WorkloadEndpoints().Update(context.Background(), wep, libcaliopt.SetOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
