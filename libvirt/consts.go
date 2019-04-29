package libvirt

import libvirtgo "github.com/libvirt/libvirt-go"

const (
	// ListAllDomainFlags shows all states for listing domain operation.
	ListAllDomainFlags = libvirtgo.CONNECT_LIST_DOMAINS_ACTIVE |
		libvirtgo.CONNECT_LIST_DOMAINS_INACTIVE |
		libvirtgo.CONNECT_LIST_DOMAINS_PERSISTENT |
		libvirtgo.CONNECT_LIST_DOMAINS_TRANSIENT |
		libvirtgo.CONNECT_LIST_DOMAINS_RUNNING |
		libvirtgo.CONNECT_LIST_DOMAINS_PAUSED |
		libvirtgo.CONNECT_LIST_DOMAINS_SHUTOFF |
		libvirtgo.CONNECT_LIST_DOMAINS_OTHER |
		libvirtgo.CONNECT_LIST_DOMAINS_MANAGEDSAVE |
		libvirtgo.CONNECT_LIST_DOMAINS_NO_MANAGEDSAVE |
		libvirtgo.CONNECT_LIST_DOMAINS_AUTOSTART |
		libvirtgo.CONNECT_LIST_DOMAINS_NO_AUTOSTART |
		libvirtgo.CONNECT_LIST_DOMAINS_HAS_SNAPSHOT |
		libvirtgo.CONNECT_LIST_DOMAINS_NO_SNAPSHOT

	// DomainNoState .
	DomainNoState = libvirtgo.DOMAIN_NOSTATE
	// DomainRunning .
	DomainRunning = libvirtgo.DOMAIN_RUNNING
	// DomainUndefineManagedSave .
	DomainUndefineManagedSave = libvirtgo.DOMAIN_UNDEFINE_MANAGED_SAVE
	// DomainShutoff is shutted down.
	DomainShutoff = libvirtgo.DOMAIN_SHUTOFF
	// DomainShutting is shuting state.
	DomainShutting = libvirtgo.DOMAIN_SHUTDOWN
	// DomainPMSuspended .
	DomainPMSuspended = libvirtgo.DOMAIN_PMSUSPENDED
	// DomainCrashed .
	DomainCrashed = libvirtgo.DOMAIN_CRASHED
	// DomainPaused .
	DomainPaused = libvirtgo.DOMAIN_PAUSED
	// DomainBlocked .
	DomainBlocked = libvirtgo.DOMAIN_BLOCKED

	// DomainDestroyDefault .
	DomainDestroyDefault = libvirtgo.DOMAIN_DESTROY_DEFAULT
	// DomainShutdownDefault .
	DomainShutdownDefault = libvirtgo.DOMAIN_SHUTDOWN_DEFAULT

	// DomainVcpuCurrent .
	DomainVcpuCurrent = libvirtgo.DOMAIN_VCPU_CURRENT
	// DomainVcpuMaximum .
	DomainVcpuMaximum = libvirtgo.DOMAIN_VCPU_MAXIMUM
	// DomainVcpuConfig .
	DomainVcpuConfig = libvirtgo.DOMAIN_VCPU_CONFIG
	// DomainVcpuLive .
	DomainVcpuLive = libvirtgo.DOMAIN_VCPU_LIVE

	// DomainMemCurrent .
	DomainMemCurrent = libvirtgo.DOMAIN_MEM_CURRENT
	// DomainMemMaximum .
	DomainMemMaximum = libvirtgo.DOMAIN_MEM_MAXIMUM
	// DomainMemConfig .
	DomainMemConfig = libvirtgo.DOMAIN_MEM_CONFIG

	// DomainConsoleForce .
	DomainConsoleForce = libvirtgo.DOMAIN_CONSOLE_FORCE
	// DomainConsoleSafe .
	DomainConsoleSafe = libvirtgo.DOMAIN_CONSOLE_SAFE

	// DomainBlockResizeBytes .
	DomainBlockResizeBytes = libvirtgo.DOMAIN_BLOCK_RESIZE_BYTES

	// DomainDeviceModifyConfig .
	DomainDeviceModifyConfig = libvirtgo.DOMAIN_DEVICE_MODIFY_CONFIG
	// DomainDeviceModifyCurrent .
	DomainDeviceModifyCurrent = libvirtgo.DOMAIN_DEVICE_MODIFY_CURRENT
	// DomainDeviceModifyLive .
	DomainDeviceModifyLive = libvirtgo.DOMAIN_DEVICE_MODIFY_LIVE
)
