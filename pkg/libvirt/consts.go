package libvirt

import (
	libvirtgo "github.com/projecteru2/yavirt/third_party/libvirt"
)

const (
	// ListAllDomainFlags shows all states for listing domain operation.
	ListAllDomainFlags = libvirtgo.ConnectListDomainsActive |
		libvirtgo.ConnectListDomainsInactive |
		libvirtgo.ConnectListDomainsPersistent |
		libvirtgo.ConnectListDomainsTransient |
		libvirtgo.ConnectListDomainsRunning |
		libvirtgo.ConnectListDomainsPaused |
		libvirtgo.ConnectListDomainsShutoff |
		libvirtgo.ConnectListDomainsOther |
		libvirtgo.ConnectListDomainsManagedsave |
		libvirtgo.ConnectListDomainsNoManagedsave |
		libvirtgo.ConnectListDomainsAutostart |
		libvirtgo.ConnectListDomainsNoAutostart |
		libvirtgo.ConnectListDomainsHasSnapshot |
		libvirtgo.ConnectListDomainsNoSnapshot

	// DomainNoState .
	DomainNoState = libvirtgo.DomainNostate
	// DomainRunning .
	DomainRunning = libvirtgo.DomainRunning
	// DomainUndefineManagedSave .
	DomainUndefineManagedSave = libvirtgo.DomainUndefineManagedSave
	// DomainShutoff is shutted down.
	DomainShutoff = libvirtgo.DomainShutoff
	// DomainShutting is shuting state.
	DomainShutting = libvirtgo.DomainShutdown
	// DomainPMSuspended .
	DomainPMSuspended = libvirtgo.DomainPmsuspended
	// DomainCrashed .
	DomainCrashed = libvirtgo.DomainCrashed
	// DomainPaused .
	DomainPaused = libvirtgo.DomainPaused
	// DomainBlocked .
	DomainBlocked = libvirtgo.DomainBlocked

	// DomainDestroyDefault .
	DomainDestroyDefault = libvirtgo.DomainDestroyDefault
	// DomainShutdownDefault .
	DomainShutdownDefault = libvirtgo.DomainShutdownDefault

	// DomainVcpuCurrent .
	DomainVcpuCurrent = libvirtgo.DomainVCPUCurrent
	// DomainVcpuMaximum .
	DomainVcpuMaximum = libvirtgo.DomainVCPUMaximum
	// DomainVcpuConfig .
	DomainVcpuConfig = libvirtgo.DomainVCPUConfig

	// DomainVcpuLive .
	DomainVcpuLive = libvirtgo.DomainVCPULive

	// DomainMemCurrent .
	DomainMemCurrent = libvirtgo.DomainMemCurrent
	// DomainMemMaximum .
	DomainMemMaximum = libvirtgo.DomainMemMaximum
	// DomainMemConfig .
	DomainMemConfig = libvirtgo.DomainMemConfig

	// DomainConsoleForce .
	DomainConsoleForce = libvirtgo.DomainConsoleForce
	// DomainConsoleSafe .
	DomainConsoleSafe = libvirtgo.DomainConsoleSafe

	// DomainBlockResizeBytes .
	DomainBlockResizeBytes = libvirtgo.DomainBlockResizeBytes

	// DomainDeviceModifyConfig .
	DomainDeviceModifyConfig = libvirtgo.DomainDeviceModifyConfig
	// DomainDeviceModifyCurrent .
	DomainDeviceModifyCurrent = libvirtgo.DomainDeviceModifyCurrent
	// DomainDeviceModifyLive .
	DomainDeviceModifyLive = libvirtgo.DomainDeviceModifyLive
)
