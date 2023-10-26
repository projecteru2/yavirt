package libvirt

import (
	libvirtgo "github.com/projecteru2/yavirt/third_party/libvirt"
)

// DomainState .
type DomainState = libvirtgo.DomainState

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
type DomainInfo = libvirtgo.DomainGetInfoRet

// DomainXMLFlags .
type DomainXMLFlags = libvirtgo.DomainXMLFlags

// DomainConsoleFlags .
type DomainConsoleFlags = libvirtgo.DomainConsoleFlags

// DomainShutdownFlags .
type DomainShutdownFlags libvirtgo.DomainShutdownFlagValues

// DomainDestroyFlags .
type DomainDestroyFlags = libvirtgo.DomainDestroyFlagsValues

// DomainUndefineFlags .
type DomainUndefineFlags = libvirtgo.DomainUndefineFlagsValues

// DomainVcpuFlags .
type DomainVcpuFlags = libvirtgo.DomainVCPUFlags

// DomainMemoryModFlags .
type DomainMemoryModFlags = libvirtgo.DomainMemoryModFlags
