package netx

import (
	"net"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestPrefixToNetmask(t *testing.T) {
	assert.Equal(t, "255.255.255.255", PrefixToNetmask(32))
	assert.Equal(t, "255.255.255.0", PrefixToNetmask(24))
	assert.Equal(t, "255.255.0.0", PrefixToNetmask(16))
	assert.Equal(t, "255.0.0.0", PrefixToNetmask(8))
	assert.Equal(t, "255.255.252.0", PrefixToNetmask(22))
	assert.Equal(t, "255.255.240.0", PrefixToNetmask(20))
}

func TestIntToIPv4(t *testing.T) {
	assert.Equal(t, "255.255.255.0", IntToIPv4(4294967040))
	assert.Equal(t, "192.168.1.1", IntToIPv4(3232235777))
	assert.Equal(t, "10.1.2.3", IntToIPv4(167838211))
	assert.Equal(t, "127.0.0.1", IntToIPv4(2130706433))
	assert.Equal(t, "255.255.255.255", IntToIPv4(4294967295))
}

func TestIPv4ToInt(t *testing.T) {
	var cases = []struct {
		out int64
		in  string
	}{
		{3232235777, "192.168.1.1"},
		{167838211, "10.1.2.3"},
		{2130706433, "127.0.0.1"},
		{4294967295, "255.255.255.255"},
	}

	for _, c := range cases {
		var i, err = IPv4ToInt(c.in)
		assert.NilErr(t, err)
		assert.Equal(t, c.out, i)
	}
}

func TestCheckIPv4(t *testing.T) {
	var checkIP = func(b byte, subnetOnes int) error {
		var ip = net.IPv4(10, 10, 10, b)
		var mask = net.CIDRMask(subnetOnes, net.IPv4len*8)

		assert.NotNil(t, mask)
		assert.Equal(t, 4, len(mask))

		return CheckIPv4(ip, mask)
	}

	assert.Err(t, checkIP(0x0, 25))
	assert.Err(t, checkIP(0x7f, 25))
	assert.Err(t, checkIP(0x80, 25))
	assert.Err(t, checkIP(0xff, 25))
	assert.NilErr(t, checkIP(0x3e, 25))
	assert.NilErr(t, checkIP(0xee, 25))

	assert.Err(t, checkIP(0x0, 26))
	assert.Err(t, checkIP(0x3f, 26))
	assert.Err(t, checkIP(0x40, 26))
	assert.Err(t, checkIP(0xff, 26))
	assert.NilErr(t, checkIP(0x3e, 26))
	assert.NilErr(t, checkIP(0xee, 26))

	assert.Err(t, checkIP(0x0, 28))
	assert.Err(t, checkIP(0xf, 28))
	assert.Err(t, checkIP(0x10, 28))
	assert.Err(t, checkIP(0xaf, 28))
	assert.Err(t, checkIP(0xb0, 28))
	assert.NilErr(t, checkIP(0x3e, 28))
	assert.NilErr(t, checkIP(0xee, 28))

	assert.Err(t, checkIP(0x0, 30))
	assert.Err(t, checkIP(0x3, 30))
	assert.Err(t, checkIP(0x4, 30))
	assert.Err(t, checkIP(0x48, 30))
	assert.Err(t, checkIP(0x4b, 30))
	assert.NilErr(t, checkIP(0x3e, 30))
	assert.NilErr(t, checkIP(0xee, 30))

	var sub22Mask = net.CIDRMask(22, net.IPv4len*8)
	assert.Err(t, CheckIPv4(net.ParseIP("10.129.12.0"), sub22Mask))
	assert.Err(t, CheckIPv4(net.ParseIP("10.129.15.255"), sub22Mask))
	assert.Err(t, CheckIPv4(net.ParseIP("10.129.179.255"), sub22Mask))
	assert.NilErr(t, CheckIPv4(net.ParseIP("10.129.15.0"), sub22Mask))
	assert.NilErr(t, CheckIPv4(net.ParseIP("10.129.253.0"), sub22Mask))
	assert.NilErr(t, CheckIPv4(net.ParseIP("10.129.253.255"), sub22Mask))
}

func TestRealAddAddr(t *testing.T) {
	var ips = []string{
		"10.129.98.213",
		"10.129.98.214",
		"10.129.98.215",
		"10.129.98.217",
	}
	for _, c := range ips {
		var _, err = IPv4ToInt(c)
		assert.NilErr(t, err)
	}
}

func TestParseCIDROrIP(t *testing.T) {
	ipn, err := ParseCIDROrIP("10.0.0.8/24")
	assert.NilErr(t, err)
	assert.True(t, ipn.IP.Equal(net.ParseIP("10.0.0.8")))

	_, err = ParseCIDROrIP("256.0.0.0/35")
	assert.Err(t, err)

	ipn, err = ParseCIDROrIP("10.0.0.15")
	assert.NilErr(t, err)
	assert.True(t, ipn.IP.Equal(net.ParseIP("10.0.0.15")))

	_, err = ParseCIDROrIP("10.256.300.0")
	assert.Err(t, err)
}

func TestIPReachable(t *testing.T) {
	v, err := IPReachable("8.8.8.8", time.Second)
	assert.Nil(t, err)
	assert.True(t, v)
}
