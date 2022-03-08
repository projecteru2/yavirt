package netx

import (
	"net"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/util"
)

// GetOutboundIP .
func GetOutboundIP(dest string) (string, error) {
	conn, err := net.Dial("udp", dest)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer conn.Close()

	laddr := conn.LocalAddr().(*net.UDPAddr)

	return laddr.IP.String(), nil
}

// PrefixToNetmask .
func PrefixToNetmask(prefix int) string {
	var p = uint(util.Min(prefix, 32))        //nolint:gomnd // ipv4 is 32bit
	var i = ((1 << p) - 1) << uint(32-prefix) //nolint
	return IntToIPv4(int64(i))
}

// IntToIPv4 .
func IntToIPv4(i64 int64) string {
	return Int2ip(i64).String()
}

// Int2ip .
func Int2ip(i64 int64) net.IP {
	ip := make(net.IP, net.IPv4len)
	for i := 0; i < len(ip); i++ {
		seg := 0xff & (i64 >> uint(24-i*8)) //nolint
		ip[i] = byte(seg)
	}
	return ip
}

// IPv4ToInt .
func IPv4ToInt(ipv4 string) (i64 int64, err error) {
	if i64 = IP2int(net.ParseIP(ipv4).To4()); i64 == 0 {
		err = errors.Annotatef(errors.ErrInvalidValue, "invalid IPv4: %s", ipv4)
	}
	return
}

// IP2int .
func IP2int(ip net.IP) (i64 int64) {
	ip = ip.To4()
	for i, seg := range ip {
		i64 |= int64(seg) << uint(24-i*8) //nolint
	}
	return
}

// ParseCIDROrIP .
func ParseCIDROrIP(s string) (*net.IPNet, error) {
	var ipn, err = ParseCIDR2(s)
	if err == nil {
		return ipn, nil
	}

	var ip = &net.IP{}
	if err = ip.UnmarshalText([]byte(s)); err != nil {
		return nil, errors.Trace(err)
	}

	var ipv4 = ip.To4()
	if ipv4 == nil {
		return nil, errors.Annotatef(errors.ErrInvalidValue, "invalid IPv4: %s", s)
	}

	return &net.IPNet{
		IP:   ipv4,
		Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8), //nolint
	}, nil
}

// ParseCIDR2 parses, and fills the real IP address to IPNet's IP rather than the subnet.
func ParseCIDR2(cidr string) (*net.IPNet, error) {
	var ip, ipn, err = ParseCIDR(cidr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ipn.IP = ip

	return ipn, nil
}

// CheckIPv4 .
func CheckIPv4(ip net.IP, mask net.IPMask) error {
	var ipv4 = ip.To4()
	if ipv4 == nil {
		return errors.Annotatef(errors.ErrInvalidValue, "invalid IPv4: %s", ip)
	}

	var bits = net.IPv4len * 8
	var subnetOnes, _ = mask.Size()
	var allone uint32 = (1 << uint32(bits-subnetOnes)) - 1

	switch dec, err := ConvIPv4ToUint32(ipv4); {
	case err != nil:
		return errors.Trace(err)

	case dec&allone == 0:
		return errors.Annotatef(errors.ErrIPv4IsNetworkNumber, "%s", ip)

	case dec&allone == allone:
		return errors.Annotatef(errors.ErrIPv4IsBroadcastAddr, "%s", ip)

	default:
		return nil
	}
}

// ConvIPv4ToUint32 .
func ConvIPv4ToUint32(ip net.IP) (dec uint32, err error) {
	var ipv4 = ip.To4()
	if ipv4 == nil {
		return dec, errors.Annotatef(errors.ErrInvalidValue, "invalid IPv4: %s", ip)
	}

	for i := 0; i < 4; i++ {
		dec |= uint32(ipv4[i]) << uint32((3-i)*8) //nolint
	}

	return dec, nil
}

// ParseCIDR .
func ParseCIDR(cidr string) (ip net.IP, ipn *net.IPNet, err error) {
	if ip, ipn, err = net.ParseCIDR(cidr); err != nil {
		err = errors.Annotatef(errors.ErrInvalidValue, "invalid CIDR: %s", cidr)
	}
	return
}
