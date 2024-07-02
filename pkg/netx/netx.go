package netx

import (
	"net"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	probing "github.com/prometheus-community/pro-bing"
)

// GetOutboundIP .
func GetOutboundIP(dest string) (string, error) {
	conn, err := net.Dial("udp", dest)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	defer conn.Close()

	laddr := conn.LocalAddr().(*net.UDPAddr) //nolint

	return laddr.IP.String(), nil
}

// PrefixToNetmask .
func PrefixToNetmask(prefix int) string {
	var p = uint(utils.Min(prefix, 32))
	var i = ((1 << p) - 1) << uint(32-prefix)
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
		seg := 0xff & (i64 >> uint(24-i*8))
		ip[i] = byte(seg)
	}
	return ip
}

// IPv4ToInt .
func IPv4ToInt(ipv4 string) (i64 int64, err error) {
	if i64 = IP2int(net.ParseIP(ipv4).To4()); i64 == 0 {
		err = errors.Wrapf(terrors.ErrInvalidValue, "invalid IPv4: %s", ipv4)
	}
	return
}

// IP2int .
func IP2int(ip net.IP) (i64 int64) {
	ip = ip.To4()
	for i, seg := range ip {
		i64 |= int64(seg) << uint(24-i*8)
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
		return nil, errors.Wrap(err, "")
	}

	var ipv4 = ip.To4()
	if ipv4 == nil {
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "invalid IPv4: %s", s)
	}

	return &net.IPNet{
		IP:   ipv4,
		Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8),
	}, nil
}

// ParseCIDR2 parses, and fills the real IP address to IPNet's IP rather than the subnet.
func ParseCIDR2(cidr string) (*net.IPNet, error) {
	var ip, ipn, err = ParseCIDR(cidr)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ipn.IP = ip

	return ipn, nil
}

// CheckIPv4 .
func CheckIPv4(ip net.IP, mask net.IPMask) error {
	var ipv4 = ip.To4()
	if ipv4 == nil {
		return errors.Wrapf(terrors.ErrInvalidValue, "invalid IPv4: %s", ip)
	}

	var bits = net.IPv4len * 8
	var subnetOnes, _ = mask.Size()
	var allone uint32 = (1 << uint32(bits-subnetOnes)) - 1

	switch dec, err := ConvIPv4ToUint32(ipv4); {
	case err != nil:
		return errors.Wrap(err, "")

	case dec&allone == 0:
		return errors.Wrapf(terrors.ErrIPv4IsNetworkNumber, "%s", ip)

	case dec&allone == allone:
		return errors.Wrapf(terrors.ErrIPv4IsBroadcastAddr, "%s", ip)

	default:
		return nil
	}
}

// ConvIPv4ToUint32 .
func ConvIPv4ToUint32(ip net.IP) (dec uint32, err error) {
	var ipv4 = ip.To4()
	if ipv4 == nil {
		return dec, errors.Wrapf(terrors.ErrInvalidValue, "invalid IPv4: %s", ip)
	}

	for i := 0; i < 4; i++ {
		dec |= uint32(ipv4[i]) << uint32((3-i)*8)
	}

	return dec, nil
}

// ParseCIDR .
func ParseCIDR(cidr string) (ip net.IP, ipn *net.IPNet, err error) {
	if ip, ipn, err = net.ParseCIDR(cidr); err != nil {
		err = errors.Wrapf(terrors.ErrInvalidValue, "invalid CIDR: %s", cidr)
	}
	return
}

func DefaultGatewayIP(ipNet *net.IPNet) (net.IP, error) {
	// Check if the IPNet is not nil
	if ipNet == nil {
		return nil, errors.New("IPNet is nil")
	}

	// Get the first IP address in the range
	firstIP := ipNet.IP

	// Increment the IP address to get the second IP
	secondIP := make(net.IP, len(firstIP))
	copy(secondIP, firstIP)
	for i := len(secondIP) - 1; i >= 0; i-- {
		secondIP[i]++
		if secondIP[i] > firstIP[i] {
			break
		}
	}

	return secondIP, nil
}

func InSubnet(ip string, subnet string) bool {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return false
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}
	return ipNet.Contains(ipAddr)
}

func IPReachable(ip string, timeout time.Duration) (bool, error) {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		return false, err
	}
	pinger.Timeout = timeout
	pinger.Count = 2
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return false, err
	}
	stats := pinger.Statistics()
	return stats.PacketsRecv > 0, nil
}
