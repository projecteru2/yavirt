package calico

import (
	"context"
	"fmt"
	"net"

	libcalinet "github.com/projectcalico/libcalico-go/lib/net"
	libcaliopt "github.com/projectcalico/libcalico-go/lib/options"
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/errors"
)

func alignFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "dry-run",
			Value: false,
		},
	}
}

func align(c *cli.Context, runtime run.Runtime) error {
	bound, err := getGatewayBoundIPs(runtime)
	if err != nil {
		return errors.Trace(err)
	}
	return alignGatewayIPs(runtime, bound, c.Bool("dry-run"))
}

func getGatewayBoundIPs(runtime run.Runtime) ([]net.IP, error) {
	if err := runtime.CalicoHandler.InitGateway("yavirt-cali-gw"); err != nil {
		return nil, errors.Trace(err)
	}

	gw := runtime.CalicoHandler.Gateway()
	addrs, err := gw.ListAddr()
	if err != nil {
		return nil, errors.Trace(err)
	}

	ips := make([]net.IP, addrs.Len())
	for i, addr := range addrs {
		ips[i] = addr.IPNet.IP
	}

	return ips, nil
}

func alignGatewayIPs(runtime run.Runtime, bound []net.IP, dryRun bool) error {
	wep := runtime.CalicoHandler.GatewayWorkloadEndpoint()

	for _, bip := range bound {
		ipn := libcalinet.IPNet{
			IPNet: net.IPNet{
				IP:   bip,
				Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8), //nolint
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

		_, err := runtime.CalicoDriver.WorkloadEndpoint().WorkloadEndpoints().Update(context.Background(), wep, libcaliopt.SetOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
