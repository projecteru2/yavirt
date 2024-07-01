package models

import (
	"net"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// IPBlocks .
type IPBlocks []*IPBlock

// Len .
func (bs IPBlocks) Len() int { return len(bs) }

// Append .
func (bs *IPBlocks) Append(block ...*IPBlock) {
	*bs = append(*bs, block...)
}

// IPBlock .
type IPBlock struct {
	*meta.Generic

	IPs *utils.Bitmap32 `json:"ips"`

	ippool *IPPool
	ipnet  *net.IPNet
}

func newIPBlock(ipp *IPPool, ipn *net.IPNet) *IPBlock {
	block := &IPBlock{
		Generic: meta.NewGeneric(),
		ippool:  ipp,
		ipnet:   ipn,
	}

	block.Status = meta.StatusRunning
	block.IPs = utils.NewBitmap32(block.ipCount())

	return block
}

// Release .
func (b *IPBlock) Release(ipn *net.IPNet) error {
	if !b.ippool.Contains(ipn) {
		return errors.Wrapf(terrors.ErrInsufficientIP, "IP %s not found", ipn.IP)
	}

	offset := b.getIPIndex(ipn.IP)
	if err := b.IPs.Unset(offset); err != nil {
		return errors.Wrapf(err, "release %d IP %s failed", offset, ipn)
	}

	if err := b.save(); err != nil {
		b.IPs.Set(offset) //nolint
		return errors.Wrapf(err, "release %d IP %s failed", offset, ipn)
	}

	return nil
}

func (b *IPBlock) isAssigned(ipn *net.IPNet) (bool, error) {
	offset := b.getIPIndex(ipn.IP)
	return b.IPs.Has(offset)
}

func (b *IPBlock) getIPIndex(ip net.IP) int {
	i64 := netx.IP2int(ip)

	i64 &= (1 << uint(32-b.MaskBits())) - 1

	i64 %= MaxBlockIPCount

	return int(i64)
}

// Assign .
func (b *IPBlock) Assign() (ipn *net.IPNet, err error) {
	b.IPs.Range(func(offset int, set bool) bool {
		if set {
			return true
		}

		ipn, err = b.assign(offset)

		return false
	})

	if err == nil && ipn == nil {
		err = errors.Wrapf(terrors.ErrInsufficientIP,
			"block %s hasn't free IP", b.ipnet)
	}

	return
}

func (b *IPBlock) assign(offset int) (*net.IPNet, error) {
	ipn := &net.IPNet{
		IP:   netx.Int2ip(b.intIP() + int64(offset)),
		Mask: b.ipnet.Mask,
	}

	if err := b.IPs.Set(offset); err != nil {
		return nil, errors.Wrapf(err, "assign %d IP %s failed", offset, ipn)
	}

	if err := b.save(); err != nil {
		b.IPs.Unset(offset) //nolint
		return nil, errors.Wrap(err, "")
	}

	return ipn, nil
}

// HasFreeIP .
func (b *IPBlock) HasFreeIP() bool {
	return b.IPs.Available()
}

func (b *IPBlock) save() error {
	return meta.Save(meta.Resources{b})
}

func (b *IPBlock) intIP() int64 {
	return netx.IP2int(b.BlockIP())
}

// MetaKey .
func (b *IPBlock) MetaKey() string {
	return meta.IPBlockKey(b.ippool.Name, b.BlockIP().String())
}

// BlockIP .
func (b *IPBlock) BlockIP() net.IP {
	return b.ipnet.IP
}

// Marshal .
func (b *IPBlock) Marshal() ([]byte, error) {
	return utils.JSONEncode(b)
}

// CIDR .
func (b *IPBlock) CIDR() string {
	return b.ipnet.String()
}

func (b *IPBlock) ipCount() int {
	return 1 << uint(b.ipBits())
}

func (b *IPBlock) ipBits() int {
	return utils.Min(8, 32-b.MaskBits())
}

// MaskBits .
func (b *IPBlock) MaskBits() (ones int) {
	ones, _ = b.ipnet.Mask.Size()
	return
}
