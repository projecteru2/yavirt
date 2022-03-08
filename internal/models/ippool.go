package model

import (
	"context"
	"fmt"
	"net"

	"go.etcd.io/etcd/clientv3"

	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/netx"
	"github.com/projecteru2/yavirt/store"
	"github.com/projecteru2/yavirt/util"
)

// IPPool .
type IPPool struct {
	*Generic

	Name  string         `json:"name"`
	Raw   string         `json:"raw"`
	CIDR  string         `json:"cidr"`
	Flags *util.Bitmap32 `json:"flags"`

	ipnet  *net.IPNet
	blocks IPBlocks

	sync bool
}

// LoadIPPool .
func LoadIPPool(name string) (*IPPool, error) {
	ipp := newIPPool(name)

	if err := meta.Load(ipp); err != nil {
		return nil, errors.Annotatef(err, "load IPPool %s failed", name)
	}

	return ipp, ipp.parse()
}

// NewIPPool .
func NewIPPool(name, cidr string) (ipp *IPPool, err error) {
	ipp = newIPPool(name)
	ipp.Raw = cidr

	err = ipp.parse()

	return
}

func newIPPool(name string) *IPPool {
	ipp := &IPPool{
		Generic: newGeneric(),
		Name:    name,
		blocks:  IPBlocks{},
		sync:    true,
	}

	ipp.Status = StatusRunning

	return ipp
}

func (ipp *IPPool) parse() (err error) {
	if _, ipp.ipnet, err = netx.ParseCIDR(ipp.Raw); err != nil {
		return errors.Annotatef(err, "parse CIDR %s failed", ipp.Raw)
	}

	switch {
	case ipp.MaskBits() > MaxMaskBits:
		return errors.Annotatef(errors.ErrTooLargeMaskBits, "at most", MaxMaskBits)
	case ipp.MaskBits() < MinMaskBits:
		return errors.Annotatef(errors.ErrTooSmallMaskBits, "at least", MinMaskBits)
	}

	ipp.CIDR = fmt.Sprintf("%s/%d", ipp.Subnet().String(), ipp.MaskBits())
	ipp.Flags = util.NewBitmap32(ipp.blockCount())

	return
}

// Assign .
func (ipp *IPPool) Assign() (ipn *net.IPNet, err error) {
	var unlock util.Unlocker
	if unlock, err = ipp.Lock(context.Background()); err != nil {
		return
	}
	defer func() {
		if ue := unlock(context.Background()); ue != nil {
			err = errors.Wrap(err, ue)
		}
	}()

	if ipp.sync {
		if err = ipp.reload(); err != nil {
			return
		}
	}

	var block *IPBlock
	if block, err = ipp.getAvailBlock(); err != nil {
		return
	}

	return block.Assign()
}

func (ipp *IPPool) getAvailBlock() (block *IPBlock, err error) {
	ipp.Flags.Range(func(offset int, set bool) bool {
		switch {
		case !set:
			block, err = ipp.spawnBlock(offset)
			return false

		case ipp.blocks[offset].HasFreeIP():
			block = ipp.blocks[offset]
			return false

		default:
			return true
		}
	})

	// there's no any available block.
	if err == nil && block == nil {
		err = errors.Annotatef(errors.ErrInsufficientIP,
			"%s CIDR %s hasn't free IP", ipp.Name, ipp.ipnet)
	}

	return
}

// Release .
func (ipp *IPPool) Release(ipn *net.IPNet) (err error) {
	if !ipp.Contains(ipn) {
		return errors.Annotatef(errors.ErrInvalidValue, "%s doesn't contain %s", ipp.Name, ipn)
	}

	var unlock util.Unlocker
	if unlock, err = ipp.Lock(context.Background()); err != nil {
		return
	}
	defer func() {
		if ue := unlock(context.Background()); ue != nil {
			err = errors.Wrap(err, ue)
		}
	}()

	if ipp.sync {
		if err = ipp.reload(); err != nil {
			return
		}
	}

	var block *IPBlock
	if block, err = ipp.getBlock(ipn); err != nil {
		return
	}

	return block.Release(ipn)
}

// IsAssigned .
func (ipp *IPPool) IsAssigned(ipn *net.IPNet) (assigned bool, err error) {
	if !ipp.Contains(ipn) {
		return false, errors.Annotatef(errors.ErrInvalidValue, "%s doesn't contain %s", ipp.Name, ipn)
	}

	var unlock util.Unlocker
	if unlock, err = ipp.Lock(context.Background()); err != nil {
		return
	}
	defer func() {
		if ue := unlock(context.Background()); ue != nil {
			err = errors.Wrap(err, ue)
		}
	}()

	if err = ipp.reload(); err != nil {
		return
	}

	var block *IPBlock
	if block, err = ipp.getBlock(ipn); err != nil {
		return
	}

	return block.isAssigned(ipn)
}

func (ipp *IPPool) getBlock(ipn *net.IPNet) (*IPBlock, error) {
	i := ipp.getBlockIndex(ipn.IP)

	if int64(ipp.blocks.Len()) <= i {
		return nil, errors.Annotatef(errors.ErrInsufficientBlocks,
			"block %s not found", netx.Int2ip(ipp.intSubnet()+i))
	}

	return ipp.blocks[i], nil
}

// Contains checks the ipn if belongs the IPPool.
func (ipp *IPPool) Contains(ipn *net.IPNet) bool {
	if !ipp.ipnet.Contains(ipn.IP) {
		return false
	}

	if len(ipp.ipnet.Mask) != len(ipn.Mask) {
		return false
	}

	for i := 0; i < len(ipn.Mask); i++ {
		if ipp.ipnet.Mask[i] != ipn.Mask[i] {
			return false
		}
	}

	return true
}

func (ipp *IPPool) spawnBlock(offset int) (block *IPBlock, err error) {
	if err = ipp.Flags.Set(offset); err != nil {
		return nil, errors.Annotatef(err, "spawn %d block %s failed",
			offset, netx.Int2ip(ipp.intSubnet()+int64(offset)))
	}

	block = newIPBlock(ipp, ipp.getBlockIPNet(offset))
	if err = ipp.save(block); err != nil {
		ipp.Flags.Unset(offset) //nolint
		return
	}

	ipp.blocks.Append(block)

	return
}

func (ipp *IPPool) reload() error {
	newOne := newIPPool(ipp.Name)
	if err := meta.Load(newOne); err != nil {
		return errors.Annotatef(err, "load IPPool %s failed", ipp.Name)
	}

	ipp.Flags = newOne.Flags

	return ipp.reloadBlocks()
}

func (ipp *IPPool) reloadBlocks() error {
	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	prefix := meta.IPBlocksPrefix(ipp.Name)

	data, vers, err := store.GetPrefix(ctx, prefix, int64(ipp.blockCount()))
	if err != nil {
		// there's no any block yet.
		if errors.Contain(err, errors.ErrKeyNotExists) {
			return nil
		}

		return errors.Annotatef(err, "get IPPool %s all blocks failed", ipp.Name)
	}

	delete(data, prefix)

	return ipp.parseBlocksBytes(data, vers)
}

func (ipp *IPPool) parseBlocksBytes(data map[string][]byte, vers map[string]int64) error {
	blocks := make(IPBlocks, len(data))

	for key, bytes := range data {
		ver, exists := vers[key]
		if !exists {
			return errors.Annotatef(errors.ErrKeyBadVersion, key)
		}

		ipn, err := ipp.parseBlockMetaKey(key)
		if err != nil {
			return errors.Annotatef(err, "parse block key %s failed", key)
		}

		block := newIPBlock(ipp, ipn)
		if err := util.JSONDecode(bytes, block); err != nil {
			return errors.Annotatef(err, "decode IPBlock bytes %s failed", bytes)
		}

		block.SetVer(ver)

		i := ipp.getBlockIndex(block.BlockIP())
		if int64(blocks.Len()) <= i {
			return errors.Annotatef(errors.ErrInsufficientBlocks, "%d block %s not found", i, ipn)
		}

		blocks[i] = block
	}

	if err := ipp.checkBlocks(blocks); err != nil {
		return errors.Trace(err)
	}

	ipp.blocks = blocks

	return nil
}

func (ipp *IPPool) checkBlocks(blocks IPBlocks) (err error) {
	ipp.Flags.Range(func(offset int, set bool) bool {
		var valid bool
		if blocks.Len() > offset {
			valid = blocks[offset] != nil
		}

		if set == valid {
			return true
		}

		err = errors.Annotatef(errors.ErrInvalidValue,
			"IPPool %s %d block %s should be spawned (%t) but not",
			ipp.ipnet, offset, ipp.getBlockIPNet(offset), set)

		return false
	})

	return
}

func (ipp *IPPool) parseBlockMetaKey(key string) (*net.IPNet, error) {
	_, raw := util.PartRight(key, "/")
	return netx.ParseCIDR2(fmt.Sprintf("%s/%d", raw, ipp.MaskBits()))
}

// Create creates a new one, but raises an error if it has been existed.
func (ipp *IPPool) Create() error {
	ipp.SetVer(0)
	return ipp.Save()
}

func (ipp *IPPool) save(block *IPBlock) error {
	if block == nil {
		return ipp.Save()
	}

	ippBytes, err := ipp.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	blockBytes, err := block.Marshal()
	if err != nil {
		return errors.Trace(err)
	}

	ops := []clientv3.Op{
		clientv3.OpPut(ipp.MetaKey(), string(ippBytes)),
		clientv3.OpPut(block.MetaKey(), string(blockBytes)),
	}

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	switch succ, err := store.BatchOperate(ctx, ops); {
	case err != nil:
		return errors.Trace(err)
	case !succ:
		return errors.Annotatef(errors.ErrBatchOperate,
			"put: %s / %s", ipp.MetaKey(), block.MetaKey())
	}

	ipp.IncrVer()
	block.IncrVer()

	return nil
}

// Marshal .
func (ipp *IPPool) Marshal() ([]byte, error) {
	return util.JSONEncode(ipp)
}

// Save .
func (ipp *IPPool) Save() error {
	return meta.Save(meta.Resources{ipp})
}

// MetaKey .
func (ipp *IPPool) MetaKey() string {
	return meta.IPPoolKey(ipp.Name)
}

func (ipp *IPPool) intSubnet() int64 {
	return netx.IP2int(ipp.Subnet())
}

// Subnet .
func (ipp *IPPool) Subnet() net.IP {
	return ipp.ipnet.IP
}

// MaskBits .
func (ipp *IPPool) MaskBits() (ones int) {
	ones, _ = ipp.ipnet.Mask.Size()
	return
}

// Lock .
func (ipp *IPPool) Lock(ctx context.Context) (util.Unlocker, error) {
	return store.Lock(ctx, meta.IPPoolLockKey(ipp.Name))
}

func (ipp *IPPool) getBlockIPNet(offset int) *net.IPNet {
	if ipp.blockCount() <= 1 {
		return ipp.ipnet
	}

	i64 := netx.IP2int(ipp.Subnet()) + int64(uint(offset)<<8) //nolint

	return &net.IPNet{
		IP:   netx.Int2ip(i64),
		Mask: net.CIDRMask(ipp.MaskBits(), net.IPv4len*8), //nolint
	}
}

func (ipp *IPPool) getBlockIndex(ip net.IP) (i int64) {
	if ipp.blockCount() <= 1 {
		return 0
	}

	i = netx.IP2int(ip)

	// Mask the ipp's subnet.
	i &= (1 << uint(32-ipp.MaskBits())) - 1 //nolint

	// Get the index, the lowest 8 bits are IP.
	i >>= 8

	return
}

func (ipp *IPPool) blockCount() int {
	return 1 << uint(ipp.blockBits())
}

func (ipp *IPPool) blockBits() int {
	bits := MaxMaskBitsForBlocks - ipp.MaskBits() + 1
	return util.Max(bits, 0)
}
