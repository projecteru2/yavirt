package model

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/projecteru2/yavirt/pkg/netx"
	storemocks "github.com/projecteru2/yavirt/store/mocks"
	"github.com/projecteru2/yavirt/test/assert"
	"github.com/projecteru2/yavirt/test/mock"
	"github.com/projecteru2/yavirt/util"
	utilmocks "github.com/projecteru2/yavirt/util/mocks"
)

func TestIPPoolBitmapCount(t *testing.T) {
	cases := []struct {
		maskSize    int
		bitmapCount int
	}{
		{23, 1},
		{22, 1},
		{21, 1},
		{20, 1},
		{19, 1},
		{18, 2},
		{17, 4},
		{16, 8},
		{15, 16},
	}

	for _, c := range cases {
		cidr := fmt.Sprintf("1.1.1.1/%d", c.maskSize)
		ipp := newTestIPP(t, cidr)
		assert.Equal(t, c.bitmapCount, len(ipp.Flags.Slices))
	}
}

func TestIPPoolMultipleBlocks(t *testing.T) {
	for size := MinMaskBits; size <= MaxMaskBitsForBlocks; size++ {
		cidr := fmt.Sprintf("1.1.1.1/%d", size)
		ipp := newTestIPP(t, cidr)
		assert.Equal(t, 1<<uint(util.Max(0, ipp.blockBits()-5)), len(ipp.Flags.Slices))
		assert.Equal(t, 1<<uint(MaxMaskBitsForBlocks-size+1), ipp.blockCount())
		assert.Equal(t, MaxMaskBitsForBlocks-size+1, ipp.blockBits())
	}
}

func TestIPPoolSpawnMultipleBlocks(t *testing.T) {
	cases := []struct {
		cidr   string
		blocks []string
	}{
		{
			"1.1.1.1/23",
			[]string{"1.1.0.0/23", "1.1.1.0/23", ""},
		},
		{
			"1.1.1.1/22",
			[]string{"1.1.0.0/22", "1.1.1.0/22", "1.1.2.0/22", "1.1.3.0/22", ""},
		},
		{
			"1.1.254.100/22",
			[]string{"1.1.252.0/22", "1.1.253.0/22", "1.1.254.0/22", "1.1.255.0/22", ""},
		},
		{
			"1.1.1.1/16",
			[]string{"1.1.0.0/16", "1.1.1.0/16"},
		},
		{
			"1.1.1.1/15",
			[]string{"1.0.0.0/15", "1.0.1.0/15", "1.0.2.0/15"},
		},
	}

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(true, nil)

	for _, c := range cases {
		ipp := newTestIPP(t, c.cidr)

		for i, exp := range c.blocks {
			block, err := ipp.spawnBlock(i)
			if len(exp) > 0 {
				assert.Nil(t, err)
				assert.NotNil(t, block)
				assert.Equal(t, exp, block.CIDR())
				assert.Equal(t, i+1, ipp.blocks.Len())
			} else {
				assert.Err(t, err)
			}
		}
	}
}

func TestIPPoolSingleBlock(t *testing.T) {
	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(true, nil)

	for size := MaxMaskBitsForBlocks + 1; size <= MaxMaskBits; size++ {
		cidr := fmt.Sprintf("1.1.1.1/%d", size)
		ipp := newTestIPP(t, cidr)
		assert.Equal(t, 1, len(ipp.Flags.Slices))
		assert.Equal(t, 1, ipp.blockCount())

		block, err := ipp.spawnBlock(0)
		assert.NilErr(t, err)
		assert.NotNil(t, block)
		assert.Equal(t, fmt.Sprintf("1.1.1.0/%d", size), block.CIDR())
		assert.Equal(t, 1, ipp.blocks.Len())
	}
}

func TestIPPoolSpawnFailedAsStoreFailure(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/23")
	assert.Equal(t, []uint32{0}, ipp.Flags.Slices)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(false, nil).Once()

	_, err := ipp.spawnBlock(0)
	assert.Err(t, err)
	assert.Equal(t, []uint32{0}, ipp.Flags.Slices)
	assert.Equal(t, 0, ipp.blocks.Len())
}

func TestIPPoolAssignFailedAsStoreFailure(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/24")
	assert.Equal(t, []uint32{0}, ipp.Flags.Slices)

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(true, nil).Once()
	store.On("NewMutex", mock.Anything).Return(mutex, nil)
	store.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("error"))

	_, err := ipp.Assign()
	assert.Err(t, err)
}

func TestIPPoolAssignSingleBlock(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/24")
	assert.Equal(t, []uint32{0}, ipp.Flags.Slices)

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(true, nil)
	store.On("NewMutex", mock.Anything).Return(mutex, nil)
	store.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	expIPs := make([]uint32, 8)

	// Assigns all IP addresses of the block.
	for i := 0; i < MaxBlockIPCount; i++ {
		ipnet, err := ipp.Assign()
		assert.NilErr(t, err)

		expIPNet := &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("1.1.1.%d", i)).To4(),
			Mask: net.CIDRMask(ipp.MaskBits(), 32),
		}
		assert.Equal(t, expIPNet, ipnet)
		assert.Equal(t, []uint32{1}, ipp.Flags.Slices)
		assert.NotNil(t, ipp.blocks[0])

		mapIndex, bitIndex := getBitmapIndex(i)
		expIPs[mapIndex] |= util.SetBitFlags[bitIndex]
		assert.Equal(t, expIPs, ipp.blocks[0].IPs.Slices)
	}

	// there's no free IP already.
	_, err := ipp.Assign()
	assert.Err(t, err)

	// Release them all.
	for i := 0; i < MaxBlockIPCount; i++ {
		ipn := &net.IPNet{
			IP:   net.ParseIP(fmt.Sprintf("1.1.1.%d", i)).To4(),
			Mask: net.CIDRMask(ipp.MaskBits(), 32),
		}
		assert.NilErr(t, ipp.Release(ipn))

		mapIndex, bitIndex := getBitmapIndex(i)
		expIPs[mapIndex] &= util.UnsetBitFlags[bitIndex]
		assert.Equal(t, expIPs, ipp.blocks[0].IPs.Slices)
	}

	ipn := &net.IPNet{
		IP:   net.ParseIP("10.10.10.10").To4(),
		Mask: net.CIDRMask(ipp.MaskBits(), 32),
	}
	assert.Err(t, ipp.Release(ipn))
}

func TestIPPoolReleaseFailedAsOutOfIndex(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/30")
	block := newIPBlock(ipp, ipp.ipnet)
	ipn := &net.IPNet{
		IP:   net.ParseIP("1.1.1.4"),
		Mask: net.CIDRMask(30, 32),
	}
	assert.Err(t, block.Release(ipn))
}

func TestIPPoolGetIPIndex(t *testing.T) {
	cases := []struct {
		ippCIDR   string
		exp       int
		blockCIDR string
		in        string
	}{
		{"1.1.1.1/30", 0, "1.1.1.0/30", "1.1.1.0"},
		{"1.1.1.1/30", 1, "1.1.1.0/30", "1.1.1.1"},
		{"1.1.1.1/30", 2, "1.1.1.0/30", "1.1.1.2"},
		{"1.1.1.1/30", 3, "1.1.1.0/30", "1.1.1.3"},
		{"1.1.1.1/23", 0, "1.1.0.0/23", "1.1.0.0"},
		{"1.1.1.1/23", 255, "1.1.0.0/23", "1.1.0.255"},
		{"1.1.1.1/23", 0, "1.1.1.0/23", "1.1.1.0"},
		{"1.1.1.1/23", 255, "1.1.1.0/23", "1.1.1.255"},
		{"1.1.1.1/16", 0, "1.1.0.0/16", "1.1.0.0"},
		{"1.1.1.1/16", 192, "1.1.255.0/16", "1.1.255.192"},
	}

	for _, c := range cases {
		_, blockIPN, err := netx.ParseCIDR(c.blockCIDR)
		assert.NilErr(t, err)

		ipp := newTestIPP(t, c.ippCIDR)
		block := newIPBlock(ipp, blockIPN)
		assert.Equal(t, c.exp, block.getIPIndex(net.ParseIP(c.in)))
	}
}

func TestIPPoolGetBlockIndex(t *testing.T) {
	cases := []struct {
		cidr string
		exp  int64
		in   string
	}{
		{"1.1.1.1/23", 0, "1.1.0.0"},
		{"1.1.1.1/23", 0, "1.1.0.192"},
		{"1.1.1.1/23", 0, "1.1.0.255"},
		{"1.1.1.1/23", 1, "1.1.1.0"},
		{"1.1.1.1/23", 1, "1.1.1.64"},
		{"1.1.1.1/23", 1, "1.1.1.255"},
		{"1.1.1.1/23", 0, "1.1.2.0"},
		{"1.1.1.1/23", 0, "1.1.2.128"},
		{"1.1.1.1/23", 0, "1.1.2.255"},
		{"1.1.1.1/23", 1, "1.1.3.0"},
		{"1.1.1.1/23", 1, "1.1.3.32"},
		{"1.1.1.1/23", 1, "1.1.3.255"},
		{"1.1.1.1/16", 255, "1.1.255.0"},
		{"1.1.1.1/16", 255, "1.1.255.32"},
		{"1.1.1.1/16", 255, "1.1.255.255"},
		{"1.1.1.1/15", 0, "1.0.0.0"},
		{"1.1.1.1/15", 0, "1.0.0.32"},
		{"1.1.1.1/15", 0, "1.0.0.255"},
		{"1.1.1.1/15", 255, "1.0.255.0"},
		{"1.1.1.1/15", 255, "1.0.255.32"},
		{"1.1.1.1/15", 255, "1.0.255.255"},
		{"1.1.1.1/15", 256, "1.1.0.0"},
		{"1.1.1.1/15", 256, "1.1.0.32"},
		{"1.1.1.1/15", 256, "1.1.0.255"},
		{"1.1.1.1/15", 511, "1.1.255.0"},
		{"1.1.1.1/15", 511, "1.1.255.32"},
		{"1.1.1.1/15", 511, "1.1.255.255"},
	}

	for _, c := range cases {
		ipp := newTestIPP(t, c.cidr)
		assert.Equal(t, c.exp, ipp.getBlockIndex(net.ParseIP(c.in)))
	}
}

func TestIPPoolAssignCrossBlocks(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/22")
	assert.Equal(t, 4, ipp.blockCount())

	expFlags := []uint32{0}
	assert.Equal(t, expFlags, ipp.Flags.Slices)

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)
	store.On("BatchOperate", mock.Anything, mock.Anything).Return(true, nil)
	store.On("NewMutex", mock.Anything).Return(mutex, nil)
	store.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	for blockIndex := 0; blockIndex < ipp.blockCount(); blockIndex++ {
		expIPs := make([]uint32, 8)

		mapIndex, bitIndex := getBitmapIndex(blockIndex)
		expFlags[mapIndex] |= util.SetBitFlags[bitIndex]

		for ipIndex := 0; ipIndex < MaxBlockIPCount; ipIndex++ {
			ipnet, err := ipp.Assign()
			assert.NilErr(t, err)

			expIPNet := &net.IPNet{
				IP:   net.ParseIP(fmt.Sprintf("1.1.%d.%d", blockIndex, ipIndex)).To4(),
				Mask: net.CIDRMask(ipp.MaskBits(), 32),
			}
			assert.Equal(t, expIPNet, ipnet)
			assert.NotNil(t, ipp.blocks[blockIndex])
			assert.Equal(t, expFlags, ipp.Flags.Slices)

			ipMapIndex, ipBitIndex := getBitmapIndex(ipIndex)
			expIPs[ipMapIndex] |= util.SetBitFlags[ipBitIndex]
			assert.Equal(t, expIPs, ipp.blocks[blockIndex].IPs.Slices)
		}
	}

	// there's no free one.
	_, err := ipp.Assign()
	assert.Err(t, err)

	for blockIndex := 1; blockIndex < ipp.blockCount(); blockIndex++ {
		expIPs := make([]uint32, 8)
		for i := 0; i < len(expIPs); i++ {
			expIPs[i] = (1 << 32) - 1
		}

		for ipIndex := 0; ipIndex < MaxBlockIPCount; ipIndex++ {
			ipn := &net.IPNet{
				IP:   net.ParseIP(fmt.Sprintf("1.1.%d.%d", blockIndex, ipIndex)),
				Mask: net.CIDRMask(ipp.MaskBits(), 32),
			}
			assert.NilErr(t, ipp.Release(ipn))

			ipMapIndex, ipBitIndex := getBitmapIndex(ipIndex)
			expIPs[ipMapIndex] &= util.UnsetBitFlags[ipBitIndex]
			assert.Equal(t, expIPs, ipp.blocks[blockIndex].IPs.Slices)
		}
	}

	ipn := &net.IPNet{
		IP:   net.ParseIP("1.1.4.1").To4(),
		Mask: net.CIDRMask(ipp.MaskBits(), 32),
	}
	assert.Err(t, ipp.Release(ipn))
}

func TestIPPoolReloadSingleBlocks(t *testing.T) {
	for size := MaxMaskBitsForBlocks + 1; size <= MaxMaskBits; size++ {
		ipp := newTestIPP(t, fmt.Sprintf("1.1.1.1/%d", size))
		ipp.Flags.Set(0)

		store, cancel := storemocks.Mock()
		defer cancel()
		defer store.AssertExpectations(t)

		key := "/prefix/1.1.1.0"
		data := map[string][]byte{key: encjson(t, IPBlock{})}
		vers := map[string]int64{key: 1}
		assert.NilErr(t, ipp.parseBlocksBytes(data, vers))
		assert.Equal(t, 1, ipp.blocks.Len())

		for expBlockIndex, b := range ipp.blocks {
			assert.Equal(t, int64(expBlockIndex), ipp.getBlockIndex(b.BlockIP()))
		}
	}
}

func TestIPPoolReloadMultipleUnspawnedBlocks(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/22")
	ipp.Flags.Set(0)
	ipp.Flags.Set(1)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)

	key0 := "/prefix/1.1.0.0"
	key1 := "/prefix/1.1.1.0"
	data := map[string][]byte{
		key0: encjson(t, IPBlock{}),
		key1: encjson(t, IPBlock{}),
	}
	vers := map[string]int64{
		key0: 1,
		key1: 1,
	}
	assert.NilErr(t, ipp.parseBlocksBytes(data, vers))
	assert.Equal(t, 2, ipp.blocks.Len())

	for expBlockIndex, b := range ipp.blocks {
		assert.Equal(t, int64(expBlockIndex), ipp.getBlockIndex(b.BlockIP()))
	}
}

func TestIPPoolReloadMultipleBlocks(t *testing.T) {
	ipp := newTestIPP(t, "1.1.1.1/23")
	ipp.Flags.Set(0)
	ipp.Flags.Set(1)

	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)

	key0 := "/prefix/1.1.0.0"
	key1 := "/prefix/1.1.1.0"
	data := map[string][]byte{
		key0: encjson(t, IPBlock{}),
		key1: encjson(t, IPBlock{}),
	}
	vers := map[string]int64{
		key0: 1,
		key1: 1,
	}
	assert.NilErr(t, ipp.parseBlocksBytes(data, vers))
	assert.Equal(t, 2, ipp.blocks.Len())

	for expBlockIndex, b := range ipp.blocks {
		assert.Equal(t, int64(expBlockIndex), ipp.getBlockIndex(b.BlockIP()))
	}
}

func TestIPPoolReloadBlocksFailedAsBlockNotSpawned(t *testing.T) {
	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)

	key0 := "/prefix/1.1.0.0"
	data := map[string][]byte{key0: encjson(t, IPBlock{})}
	vers := map[string]int64{key0: 1}

	ipp := newTestIPP(t, "1.1.1.1/22")
	assert.Err(t, ipp.parseBlocksBytes(data, vers))
}

func TestIPPoolReloadBlocksFailedAsBlockSpawned(t *testing.T) {
	store, cancel := storemocks.Mock()
	defer cancel()
	defer store.AssertExpectations(t)

	key0 := "/prefix/1.1.0.0"
	key1 := "/prefix/1.1.1.0"
	data := map[string][]byte{
		key0: encjson(t, IPBlock{}),
		key1: encjson(t, IPBlock{}),
	}
	vers := map[string]int64{
		key0: 1,
		key1: 1,
	}

	ipp := newTestIPP(t, "1.1.1.1/22")
	assert.Err(t, ipp.parseBlocksBytes(data, vers))
}

func TestIPPoolMultipleBlocksBitmapCountAlways8(t *testing.T) {
	for size := MinMaskBits; size <= MaxMaskBitsForBlocks; size++ {
		cidr := fmt.Sprintf("1.1.1.1/%d", size)
		ipp := newTestIPP(t, cidr)

		_, blockIPN, err := netx.ParseCIDR(fmt.Sprintf("1.1.0.0/%d", size))
		assert.NilErr(t, err)

		block := newIPBlock(ipp, blockIPN)
		assert.Equal(t, 8, len(block.IPs.Slices))
	}
}

func TestIPPoolSingleBlockBitmapCount(t *testing.T) {
	cases := []struct {
		maskSize int
		exp      int
	}{
		{24, 8},
		{25, 4},
		{26, 2},
		{27, 1},
		{28, 1},
		{30, 1},
	}

	for _, c := range cases {
		cidr := fmt.Sprintf("1.1.1.1/%d", c.maskSize)
		ipp := newTestIPP(t, cidr)

		_, blockIPN, err := netx.ParseCIDR(fmt.Sprintf("1.1.1.0/%d", c.maskSize))
		assert.NilErr(t, err)

		block := newIPBlock(ipp, blockIPN)
		assert.Equal(t, c.exp, len(block.IPs.Slices))
	}
}

func mockMutex() *utilmocks.Locker {
	var unlock util.Unlocker = func(context.Context) error {
		return nil
	}

	mutex := &utilmocks.Locker{}
	mutex.On("Lock", mock.Anything).Return(unlock, nil)

	return mutex
}

func newTestIPP(t *testing.T, cidr string) *IPPool {
	ipp, err := NewIPPool("foo", cidr)
	assert.NilErr(t, err)
	assert.NotNil(t, ipp)

	ipp.sync = false

	return ipp
}

func encjson(t *testing.T, v interface{}) []byte {
	bytes, err := util.JSONEncode(v)
	assert.NilErr(t, err)
	return bytes
}

func getBitmapIndex(offset int) (mi, bi int) {
	bm := util.NewBitmap32(256)
	return bm.GetIndex(offset)
}
