package calico

import (
	"context"
	"net"
	"sync"

	libcaliipam "github.com/projectcalico/calico/libcalico-go/lib/ipam"
	libcalinet "github.com/projectcalico/calico/libcalico-go/lib/net"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Ipam .
type Ipam struct {
	*Driver
	lck sync.Mutex
}

func newIpam(driver *Driver) *Ipam {
	return &Ipam{Driver: driver}
}

// Assign .
func (ipam *Ipam) Assign(_ context.Context, args *types.EndpointArgs) (meta.IP, error) {
	hn := configs.Hostname()

	ipam.lck.Lock()
	defer ipam.lck.Unlock()

	ipn, err := ipam.getIPv4Net(args.Calico.IPPool)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	caliArgs := libcaliipam.AutoAssignArgs{
		Num4:        1,
		Hostname:    hn,
		IPv4Pools:   []libcalinet.IPNet{*ipn},
		IntendedUse: "Workload",
	}

	return ipam.assign(caliArgs)
}

func (ipam *Ipam) assign(args libcaliipam.AutoAssignArgs) (meta.IP, error) {
	var ipv4s, err = ipam.autoAssign(args)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if len(ipv4s.IPs) < 1 {
		return nil, errors.Wrap(terrors.ErrInsufficientIP, "")
	}

	var ip = ipv4s.IPs[0]
	var ones, _ = ip.Mask.Size()
	if ones >= 30 {
		return nil, errors.Wrapf(terrors.ErrCalicoTooSmallSubnet, "/%d", ones)
	}

	if err := netx.CheckIPv4(ip.IP, ip.Mask); err != nil {
		if !terrors.IsIPv4IsNetworkNumberErr(err) && !terrors.IsIPv4IsBroadcastErr(err) {
			return nil, errors.Wrap(err, "")
		}

		// Occupies the network no. and broadcast addr.,
		// doesn't release them to Calico unallocated pool.
		// and then retry to assign.
		log.Warnf(context.TODO(), "occupy %s as it's a network no. or broadcast addr.", ip)
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

func (ipam *Ipam) getIPv4Net(poolName string) (*libcalinet.IPNet, error) {
	pool, err := ipam.getIPPool(poolName)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	_, ipn, err := libcalinet.ParseCIDR(pool.Spec.CIDR)

	switch {
	case err != nil:
		return nil, errors.Wrap(err, pool.Spec.CIDR)

	case ipn.Version() != CalicoIPv4Version:
		return nil, errors.Wrapf(terrors.ErrCalicoIPv4Only, "%d", ipn.Version())
	}

	return ipn, err
}

// Release .
func (ipam *Ipam) Release(ctx context.Context, ips ...meta.IP) error {
	ipam.lck.Lock()
	defer ipam.lck.Unlock()

	var releaseOpts = make([]libcaliipam.ReleaseOptions, len(ips))
	for i := range ips {
		var ip, ok = ips[i].(*IP)
		if !ok {
			return errors.Wrapf(terrors.ErrInvalidValue, "expect *IP, but %v", ips[i])
		}

		releaseOpts[i] = libcaliipam.ReleaseOptions{Address: ip.IP.String()}
	}

	return etcd.RetryTimedOut(func() error {
		var _, err = ipam.IPAM().ReleaseIPs(ctx, releaseOpts...)
		return err
	}, 3) //nolint:gomnd // try 3 times
}

// Query .
func (ipam *Ipam) Query(ctx context.Context, args meta.IPNets) ([]meta.IP, error) {
	ipam.lck.Lock()
	defer ipam.lck.Unlock()

	var ips = make([]meta.IP, len(args))
	var err error

	for i := range args {
		if ips[i], err = ipam.load(ctx, args[i]); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	return ips, nil
}

func (ipam *Ipam) load(_ context.Context, arg *meta.IPNet) (*IP, error) {
	var ip, err = ParseCIDR(arg.CIDR())
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	gwIPNet, err := ipam.getGatewayIPNet(arg)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ip.BindGatewayIPNet(gwIPNet)

	return ip, nil
}

func (ipam *Ipam) getGatewayIPNet(arg *meta.IPNet) (*net.IPNet, error) {
	return netx.ParseCIDR2(arg.GatewayCIDR())
}

// NewIP .
func (h *Driver) NewIP(_, cidr string) (meta.IP, error) {
	return ParseCIDR(cidr)
}

// AssignIP .
func (h *Driver) AssignIP(args *types.EndpointArgs) (ip meta.IP, err error) {
	h.Lock()
	defer h.Unlock()
	return h.assignIP(args)
}

func (h *Driver) assignIP(args *types.EndpointArgs) (ip meta.IP, err error) {
	if ip, err = h.ipam().Assign(context.TODO(), args); err != nil {
		return nil, errors.Wrap(err, "")
	}

	var roll = ip
	defer func() {
		if err != nil && roll != nil {
			if re := h.releaseIPs(roll); re != nil {
				err = errors.CombineErrors(err, re)
			}
		}
	}()

	_, gwIPNet, err := net.ParseCIDR("169.254.1.1/32")
	ip.BindGatewayIPNet(gwIPNet)
	return ip, err
}

// ReleaseIPs .
func (h *Driver) ReleaseIPs(ips ...meta.IP) error {
	h.Lock()
	defer h.Unlock()
	return h.releaseIPs(ips...)
}

func (h *Driver) releaseIPs(ips ...meta.IP) error {
	return h.ipam().Release(context.Background(), ips...)
}

// QueryIPs .
func (h *Driver) QueryIPs(ipns meta.IPNets) ([]meta.IP, error) {
	return h.ipam().Query(context.Background(), ipns)
}

func (h *Driver) ipam() network.Ipam {
	return h.Ipam()
}

// QueryIPv4 .
func (h *Driver) QueryIPv4(_ string) (meta.IP, error) {
	return nil, errors.Wrap(terrors.ErrNotImplemented, "QueryIPv4 error")
}
