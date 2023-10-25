package libvirt

import (
	golibvirt "github.com/projecteru2/yavirt/third_party/libvirt"
)

const (
	// ListAllDomainFlags shows all states for listing domain operation.
	ListAllDomainFlags = golibvirt.ConnectListDomainsActive |
		golibvirt.ConnectListDomainsInactive |
		golibvirt.ConnectListDomainsPersistent |
		golibvirt.ConnectListDomainsTransient |
		golibvirt.ConnectListDomainsRunning |
		golibvirt.ConnectListDomainsPaused |
		golibvirt.ConnectListDomainsShutoff |
		golibvirt.ConnectListDomainsOther |
		golibvirt.ConnectListDomainsManagedsave |
		golibvirt.ConnectListDomainsNoManagedsave |
		golibvirt.ConnectListDomainsAutostart |
		golibvirt.ConnectListDomainsNoAutostart |
		golibvirt.ConnectListDomainsHasSnapshot |
		golibvirt.ConnectListDomainsNoSnapshot

	// DomainNoState .
	DomainNoState = golibvirt.DomainNostate
	// DomainRunning .
	DomainRunning = golibvirt.DomainRunning
	// DomainUndefineManagedSave .
	DomainUndefineManagedSave = golibvirt.DomainUndefineManagedSave
	// DomainShutoff is shutted down.
	DomainShutoff = golibvirt.DomainShutoff
	// DomainShutting is shuting state.
	DomainShutting = golibvirt.DomainShutdown
	// DomainPMSuspended .
	DomainPMSuspended = golibvirt.DomainPmsuspended
	// DomainCrashed .
	DomainCrashed = golibvirt.DomainCrashed
	// DomainPaused .
	DomainPaused = golibvirt.DomainPaused
	// DomainBlocked .
	DomainBlocked = golibvirt.DomainBlocked

	// DomainDestroyDefault .
	DomainDestroyDefault = golibvirt.DomainDestroyDefault
	// DomainShutdownDefault .
	DomainShutdownDefault = golibvirt.DomainShutdownDefault

	// DomainVcpuCurrent .
	DomainVcpuCurrent = golibvirt.DomainVCPUCurrent
	// DomainVcpuMaximum .
	DomainVcpuMaximum = golibvirt.DomainVCPUMaximum
	// DomainVcpuConfig .
	DomainVcpuConfig = golibvirt.DomainVCPUConfig

	// DomainVcpuLive .
	DomainVcpuLive = golibvirt.DomainVCPULive

	// DomainMemCurrent .
	DomainMemCurrent = golibvirt.DomainMemCurrent
	// DomainMemMaximum .
	DomainMemMaximum = golibvirt.DomainMemMaximum
	// DomainMemConfig .
	DomainMemConfig = golibvirt.DomainMemConfig

	// DomainConsoleForce .
	DomainConsoleForce = golibvirt.DomainConsoleForce
	// DomainConsoleSafe .
	DomainConsoleSafe = golibvirt.DomainConsoleSafe

	// DomainBlockResizeBytes .
	DomainBlockResizeBytes = golibvirt.DomainBlockResizeBytes

	// DomainDeviceModifyConfig .
	DomainDeviceModifyConfig = golibvirt.DomainDeviceModifyConfig
	// DomainDeviceModifyCurrent .
	DomainDeviceModifyCurrent = golibvirt.DomainDeviceModifyCurrent
	// DomainDeviceModifyLive .
	DomainDeviceModifyLive = golibvirt.DomainDeviceModifyLive
)
