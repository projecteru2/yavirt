package calico

import "net"

const (
	// OrchestratorID .
	OrchestratorID = "yavirt"

	// CalicoIPv4Version .
	CalicoIPv4Version = 4
)

// AllonesMask .
var AllonesMask = net.CIDRMask(32, net.IPv4len*8)
