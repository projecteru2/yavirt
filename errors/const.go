package errors

var (
	// ErrInvalidValue indicates the value is invalid.
	ErrInvalidValue = New("invalid value")

	// ErrExecIsRunning .
	ErrExecIsRunning = New("exec is still running")
	// ErrExecNonZeroReturn .
	ErrExecNonZeroReturn = New("exec return code is non-zero")
	// ErrExecOnNonRunningGuest .
	ErrExecOnNonRunningGuest = New("exec on a non-running guest")

	// ErrIPv4IsNetworkNumber .
	ErrIPv4IsNetworkNumber = New("IPv4 is a network number")
	// ErrIPv4IsBroadcastAddr .
	ErrIPv4IsBroadcastAddr = New("IPv4 is a broadcast addr")

	// ErrCalicoEndpointNotExists .
	ErrCalicoEndpointNotExists = New("Calico WorkloadEndpoint not exists")
	// ErrCalicoPoolNotExists .
	ErrCalicoPoolNotExists = New("Calico IP pool not exists")
	// ErrCalicoIPv4Only .
	ErrCalicoIPv4Only = New("only support Calico IPv4")
	// ErrCalicoCannotCrossBlocks .
	ErrCalicoCannotCrossBlocks = New("cannot cross Calico blocks")
	// ErrCalicoGatewayIPNotExists .
	ErrCalicoGatewayIPNotExists = New("Calico gateway IP not exists")
	// ErrCalicoTooSmallSubnet .
	ErrCalicoTooSmallSubnet = New("Calico subnet is too small to use")

	// ErrVirtLinkExists .
	ErrVirtLinkExists = New("link exists")
	// ErrVirtLinkNotExists .
	ErrVirtLinkNotExists = New("link not exists")
	// ErrVirtLinkAddrExists .
	ErrVirtLinkAddrExists = New("link addr exists")
	// ErrVirtLinkRouteExists .
	ErrVirtLinkRouteExists = New("link route exists")

	// ErrKeyExists .
	ErrKeyExists = New("key exists")
	// ErrKeyNotExists .
	ErrKeyNotExists = New("key not exists")
	// ErrKeyBadVersion .
	ErrKeyBadVersion = New("bad version")

	// ErrBatchOperate .
	ErrBatchOperate = New("batch operate error")
	// ErrOperateIP .
	ErrOperateIP = New("operate IP error")
	// ErrForwardStatus .
	ErrForwardStatus = New("cannot forward status")
	// ErrConnectLibvirtd .
	ErrConnectLibvirtd = New("connect libvirtd error")
	// ErrSysVolumeNotExists .
	ErrSysVolumeNotExists = New("sys volume not exists")
	// ErrNotSysVolume .
	ErrNotSysVolume = New("not sys volume")
	// ErrTooManyVolumes .
	ErrTooManyVolumes = New("too many extra volumes")
	// ErrCannotShrinkVolume .
	ErrCannotShrinkVolume = New("cannot shrink a volume")
	// ErrDomainNotExists .
	ErrDomainNotExists = New("domain not exists")

	// ErrTooLargeOffset .
	ErrTooLargeOffset = New("too large offset")
	// ErrNoSuchIPPool .
	ErrNoSuchIPPool = New("no such IPPool")
	// ErrTooLargeMaskBits .
	ErrTooLargeMaskBits = New("too large mask bits")
	// ErrTooSmallMaskBits .
	ErrTooSmallMaskBits = New("too small mask bits")
	// ErrInsufficientBlocks .
	ErrInsufficientBlocks = New("insufficient blocks")
	// ErrInsufficientIP .
	ErrInsufficientIP = New("insufficient free IP")
	// ErrIPIsnotAssigned .
	ErrIPIsnotAssigned = New("IP isn't assigned")

	// ErrSerializedTaskAborted .
	ErrSerializedTaskAborted = New("serialized task was aborted in advance")

	// ErrTimeout .
	ErrTimeout = New("timed out")
	// ErrNotImplemented .
	ErrNotImplemented = New("does not implemented")

	// ErrFolderExists .
	ErrFolderExists = New("destination folder exists")
	// ErrDestinationInvalid .
	ErrDestinationInvalid = New("destination path not valid")
	// ErrNotValidCopyStatus .
	ErrNotValidCopyStatus = New("cannot copy in this status")
	// ErrNotValidLogStatus .
	ErrNotValidLogStatus = New("cannot read log in this status")

	// ErrImageHubNotConfigured .
	ErrImageHubNotConfigured = New("ImageHub is not set")
	// ErrImageFileNotExists .
	ErrImageFileNotExists = New("Image File not exists")
)
