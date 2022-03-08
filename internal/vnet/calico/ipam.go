package calico

import (
	"context"
	"net"

	libcaliipam "github.com/projectcalico/libcalico-go/lib/ipam"
	libcalinet "github.com/projectcalico/libcalico-go/lib/net"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Ipam .
type Ipam struct {
	*Driver
}

func newIpam(driver *Driver) *Ipam {
	return &Ipam{Driver: driver}
}

// Assign .
func (ipam *Ipam) Assign(ctx context.Context) (meta.IP, error) {
	hn, err := utils.Hostname()
	if err != nil {
		return nil, errors.Trace(err)
	}

	ipam.Lock()
	defer ipam.Unlock()

	ipn, err := ipam.getIPv4Net()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var args = libcaliipam.AutoAssignArgs{
		Num4:        1,
		Hostname:    hn,
		IPv4Pools:   []libcalinet.IPNet{*ipn},
		IntendedUse: "Workload",
	}

	return ipam.assign(args)
}

func (ipam *Ipam) assign(args libcaliipam.AutoAssignArgs) (meta.IP, error) {
	var ipv4s, err = ipam.autoAssign(args)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(ipv4s.IPs) < 1 {
		return nil, errors.Trace(errors.ErrInsufficientIP)
	}

	var ip = ipv4s.IPs[0]
	var ones, _ = ip.Mask.Size()
	if ones >= 30 { //nolint
		return nil, errors.Annotatef(errors.ErrCalicoTooSmallSubnet, "/%d", ones)
	}

	if err := netx.CheckIPv4(ip.IP, ip.Mask); err != nil {
		if !errors.IsIPv4IsNetworkNumberErr(err) && !errors.IsIPv4IsBroadcastErr(err) {
			return nil, errors.Trace(err)
		}

		// Occupies the network no. and broadcast addr.,
		// doesn't release them to Calico unallocated pool.
		// and then retry to assign.
		log.Warnf("occupy %s as it's a network no. or broadcast addr.", ip)
		return ipam.assign(args)
	}

	return NewIP(&ip), nil
}

func (ipam *Ipam) autoAssign(args libcaliipam.AutoAssignArgs) (ipv4s *libcaliipam.IPAMAssignments, err error) {
	etcd.RetryTimedOut(func() error { //nolint
		ipv4s, _, err = ipam.IPAM().AutoAssign(context.Background(), args)
		return err
	}, 3) //nolint:gomnd // try 3 times
	return
}

func (ipam *Ipam) getIPv4Net() (*libcalinet.IPNet, error) {
	pool, err := ipam.getIPPool()
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, ipn, err := libcalinet.ParseCIDR(pool.Spec.CIDR)

	switch {
	case err != nil:
		return nil, errors.Annotatef(err, pool.Spec.CIDR)

	case ipn.Version() != CalicoIPv4Version:
		return nil, errors.Annotatef(errors.ErrCalicoIPv4Only, "%d", ipn.Version())
	}

	return ipn, err
}

// Release .
func (ipam *Ipam) Release(ctx context.Context, ips ...meta.IP) error {
	ipam.Lock()
	defer ipam.Unlock()

	var caliIPs = make([]libcalinet.IP, len(ips))
	for i := range ips {
		var ip, ok = ips[i].(*IP)
		if !ok {
			return errors.Annotatef(errors.ErrInvalidValue, "expect *IP, but %v", ips[i])
		}

		caliIPs[i] = libcalinet.IP{IP: ip.IP}
	}

	return etcd.RetryTimedOut(func() error {
		var _, err = ipam.IPAM().ReleaseIPs(ctx, caliIPs)
		return err
	}, 3) //nolint:gomnd // try 3 times
}

// Query .
func (ipam *Ipam) Query(ctx context.Context, args meta.IPNets) ([]meta.IP, error) {
	ipam.Lock()
	defer ipam.Unlock()

	var ips = make([]meta.IP, len(args))
	var err error

	for i := range args {
		if ips[i], err = ipam.load(ctx, args[i]); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return ips, nil
}

func (ipam *Ipam) load(_ context.Context, arg *meta.IPNet) (*IP, error) {
	var ip, err = ParseCIDR(arg.CIDR())
	if err != nil {
		return nil, errors.Trace(err)
	}

	gwIPNet, err := ipam.getGatewayIPNet(arg)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ip.BindGatewayIPNet(gwIPNet)

	return ip, nil
}

func (ipam *Ipam) getGatewayIPNet(arg *meta.IPNet) (*net.IPNet, error) {
	return netx.ParseCIDR2(arg.GatewayCIDR())
}
