package calico

import "net"

// AllonesMask .
var AllonesMask = net.CIDRMask(32, net.IPv4len*8) //nolint
