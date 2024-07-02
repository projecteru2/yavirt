package network

const (
	// CalicoMode .
	CalicoMode = "calico"
	// Network CalicoCNI
	CalicoCNIMode = "calico-cni"
	// VlanMode .
	VlanMode = "vlan"
	// OVNMode
	OVNMode = "ovn"
	// FakeMode
	FakeMode = "fake"

	ModeLabelKey   = "network/mode"
	CalicoLabelKey = "network/calico"
	OVNLabelKey    = "network/ovn"
)
