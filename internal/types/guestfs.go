package types

import guestfstypes "github.com/projecteru2/yavirt/internal/virt/guestfs/types"

const (
	// FstabFile .
	FstabFile = guestfstypes.FstabFile
	// EthUbuntuFileFmt .
	EthUbuntuFileFmt = "/etc/network/interfaces.d/%s.cfg"
	// EthCentOSFileFmt .
	EthCentOSFileFmt = "/etc/sysconfig/network-scripts/ifcfg-%s"
)
