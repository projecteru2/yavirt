package libvirt

import (
	golibvirt "github.com/projecteru2/yavirt/third_party/libvirt"
)

// DomainState .
type DomainState = golibvirt.DomainState

// GetDomainStatesStrings .
func GetDomainStatesStrings(ss []DomainState) []string {
	strs := make([]string, len(ss))
	for i, s := range ss {
		strs[i] = GetDomainStateString(s)
	}
	return strs
}

// GetDomainStateString .
func GetDomainStateString(s DomainState) string {
	switch s {
	case DomainRunning:
		return "running"

	case DomainBlocked:
		return "blocked"

	case DomainPaused:
		return "paused"

	case DomainShutting:
		return "shutdowning"

	case DomainCrashed:
		return "crashed"

	case DomainPMSuspended:
		return "pmsuspended"

	case DomainShutoff:
		return "shutoff"

	case DomainNoState:
		fallthrough
	default:
		return "nostate"
	}
}

// DomainInfo .
type DomainInfo = golibvirt.DomainGetInfoRet

// DomainXMLFlags .
type DomainXMLFlags = golibvirt.DomainXMLFlags

// DomainConsoleFlags .
type DomainConsoleFlags = golibvirt.DomainConsoleFlags

// DomainShutdownFlags .
type DomainShutdownFlags golibvirt.DomainShutdownFlagValues

// DomainDestroyFlags .
type DomainDestroyFlags = golibvirt.DomainDestroyFlagsValues

// DomainUndefineFlags .
type DomainUndefineFlags = golibvirt.DomainUndefineFlagsValues

// DomainVcpuFlags .
type DomainVcpuFlags = golibvirt.DomainVCPUFlags

// DomainMemoryModFlags .
type DomainMemoryModFlags = golibvirt.DomainMemoryModFlags
