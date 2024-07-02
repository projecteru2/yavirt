package vlan

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Ipam .
type Ipam struct {
	subnet int64
}

// NewIpam .
func NewIpam(subnet int64) *Ipam {
	return &Ipam{
		subnet: subnet,
	}
}

// Assign .
func (ipam *Ipam) Assign(ctx context.Context, args *types.EndpointArgs) (ip meta.IP, err error) {
	var unlock utils.Unlocker
	if unlock, err = ipam.Lock(ctx); err != nil {
		return nil, errors.Wrap(err, "")
	}

	defer func() {
		if ue := unlock(context.TODO()); ue != nil {
			err = errors.CombineErrors(err, ue)
		}
	}()

	return ipam.assign(ctx, args.GuestID)
}

// Release .
func (ipam *Ipam) Release(ctx context.Context, ips ...meta.IP) (err error) {
	if err = ipam.check(ips...); err != nil {
		return
	}

	var unlock utils.Unlocker
	if unlock, err = ipam.Lock(ctx); err != nil {
		return errors.Wrap(err, "")
	}

	defer func() {
		if ue := unlock(context.Background()); ue != nil {
			err = errors.CombineErrors(err, ue)
		}
	}()

	return ipam.release(ctx, ips...)
}

// Insert .
func (ipam *Ipam) Insert(ctx context.Context, ip *IP) (err error) {
	if err = ipam.check(ip); err != nil {
		return
	}

	var unlock utils.Unlocker
	if unlock, err = ipam.Lock(ctx); err != nil {
		return errors.Wrap(err, "")
	}

	defer func() {
		if ue := unlock(context.Background()); ue != nil {
			err = errors.CombineErrors(err, ue)
		}
	}()

	var exists bool
	switch exists, err = ipam.exists(ctx, ip); {
	case err != nil:
		return errors.Wrap(err, "")
	case exists:
		return errors.Wrapf(terrors.ErrKeyExists, ip.CIDR())
	}

	return ipam.insert(ip)
}

// Query .
func (ipam *Ipam) Query(ctx context.Context, args meta.IPNets) ([]meta.IP, error) {
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
	subn, err := LoadSubnet(arg.IntSubnet)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var ip = NewIP()
	ip.Subnet = subn
	ip.Value = arg.IntIP
	ip.occupied = arg.Assigned

	if err := meta.Load(ip); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return ip, nil
}

func (ipam *Ipam) insert(ip *IP) error {
	return meta.Create(meta.Resources{ip})
}

func (ipam *Ipam) check(ips ...meta.IP) error {
	for _, ip := range ips {
		if ip.IntSubnet() != ipam.subnet {
			return errors.Wrapf(terrors.ErrInvalidValue, "invalid subnet: %s", ip.SubnetAddr())
		}
	}
	return nil
}

func (ipam *Ipam) exists(ctx context.Context, ip *IP) (bool, error) {
	var keys = []string{ip.freeKey(), ip.occupiedKey()}
	var exists, err = store.Exists(ctx, keys)
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	for _, v := range exists {
		if v {
			return true, nil
		}
	}

	return false, nil
}

func (ipam *Ipam) release(ctx context.Context, ips ...meta.IP) error {
	for _, ip := range ips {
		var iip = ip.IntIP()
		var putkey = meta.FreeIPKey(ipam.subnet, iip)
		var delkey = meta.OccupiedIPKey(ipam.subnet, iip)

		ip.BindGuestID("")

		if err := ipam.doop(ctx, ip, putkey, delkey); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func (ipam *Ipam) assign(ctx context.Context, guestID string) (*IP, error) {
	var ip, err = ipam.pickup(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ip.GuestID = guestID
	if err := ipam.occupy(ctx, ip); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return ip, nil
}

func (ipam *Ipam) occupy(ctx context.Context, ip *IP) error {
	var putkey = meta.OccupiedIPKey(ipam.subnet, ip.Value)
	var delkey = meta.FreeIPKey(ipam.subnet, ip.Value)
	return ipam.doop(ctx, ip, putkey, delkey)
}

func (ipam *Ipam) doop(ctx context.Context, ip meta.IP, putkey, delkey string) error {
	var enc, err = utils.JSONEncode(ip, "\t")
	if err != nil {
		return errors.Wrap(err, "")
	}

	var ops = []clientv3.Op{
		clientv3.OpPut(putkey, string(enc)),
		clientv3.OpDelete(delkey),
	}

	switch succ, err := store.BatchOperate(ctx, ops); {
	case err != nil:
		return errors.Wrap(err, "")
	case !succ:
		return errors.Wrapf(terrors.ErrOperateIP, "put: %s, del: %s", putkey, delkey)
	}

	return nil
}

func (ipam *Ipam) pickup(ctx context.Context) (*IP, error) {
	var subnet = NewSubnet(ipam.subnet)
	if err := meta.Load(subnet); err != nil {
		return nil, errors.Wrap(err, "")
	}

	var data, vers, err = store.GetPrefix(ctx, meta.FreeIPPrefix(ipam.subnet), 1)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var ip = NewIP()
	ip.Subnet = subnet

	for key, val := range data {
		var ver, exists = vers[key]
		if !exists {
			return nil, errors.Wrapf(terrors.ErrKeyBadVersion, key)
		}

		if err := utils.JSONDecode(val, ip); err != nil {
			return nil, errors.Wrap(err, "")
		}

		ip.SetVer(ver)
	}

	return ip, nil
}

// Lock .
func (ipam *Ipam) Lock(ctx context.Context) (utils.Unlocker, error) {
	return store.Lock(ctx, meta.IPALocKey(ipam.subnet))
}
