package terrors

import "github.com/cockroachdb/errors"

var (
	// ErrInvalidValue indicates the value is invalid.
	ErrInvalidValue = errors.New("invalid value")

	// ErrExecIsRunning .
	ErrExecIsRunning = errors.New("exec is still running")
	// ErrExecNonZeroReturn .
	ErrExecNonZeroReturn = errors.New("exec return code is non-zero")
	// ErrExecOnNonRunningGuest .
	ErrExecOnNonRunningGuest = errors.New("exec on a non-running guest")

	// ErrIPv4IsNetworkNumber .
	ErrIPv4IsNetworkNumber = errors.New("IPv4 is a network number")
	// ErrIPv4IsBroadcastAddr .
	ErrIPv4IsBroadcastAddr = errors.New("IPv4 is a broadcast addr")

	// ErrCalicoEndpointNotExists .
	ErrCalicoEndpointNotExists = errors.New("Calico WorkloadEndpoint not exists")
	// ErrCalicoPoolNotExists .
	ErrCalicoPoolNotExists = errors.New("Calico IP pool not exists")
	// ErrCalicoIPv4Only .
	ErrCalicoIPv4Only = errors.New("only support Calico IPv4")
	// ErrCalicoCannotCrossBlocks .
	ErrCalicoCannotCrossBlocks = errors.New("cannot cross Calico blocks")
	// ErrCalicoGatewayIPNotExists .
	ErrCalicoGatewayIPNotExists = errors.New("Calico gateway IP not exists")
	// ErrCalicoTooSmallSubnet .
	ErrCalicoTooSmallSubnet = errors.New("Calico subnet is too small to use")

	// ErrVirtLinkExists .
	ErrVirtLinkExists = errors.New("link exists")
	// ErrVirtLinkNotExists .
	ErrVirtLinkNotExists = errors.New("link not exists")
	// ErrVirtLinkAddrExists .
	ErrVirtLinkAddrExists = errors.New("link addr exists")
	// ErrVirtLinkRouteExists .
	ErrVirtLinkRouteExists = errors.New("link route exists")

	// ErrKeyExists .
	ErrKeyExists = errors.New("key exists")
	// ErrKeyNotExists .
	ErrKeyNotExists = errors.New("key not exists")
	// ErrKeyBadVersion .
	ErrKeyBadVersion = errors.New("bad version")

	// ErrBatchOperate .
	ErrBatchOperate = errors.New("batch operate error")
	// ErrOperateIP .
	ErrOperateIP = errors.New("operate IP error")
	// ErrForwardStatus .
	ErrForwardStatus = errors.New("cannot forward status")
	// ErrConnectLibvirtd .
	ErrConnectLibvirtd = errors.New("connect libvirtd error")
	// ErrSysVolumeNotExists .
	ErrSysVolumeNotExists = errors.New("sys volume not exists")
	// ErrNotSysVolume .
	ErrNotSysVolume = errors.New("not sys volume")
	// ErrTooManyVolumes .
	ErrTooManyVolumes = errors.New("too many extra volumes")
	// ErrCannotShrinkVolume .
	ErrCannotShrinkVolume = errors.New("cannot shrink a volume")
	// ErrDomainNotExists .
	ErrDomainNotExists = errors.New("domain not exists")
	// ErrInvalidVolumeBind .
	ErrInvalidVolumeBind = errors.New("invalid volume bind")

	// ErrTooLargeOffset .
	ErrTooLargeOffset = errors.New("too large offset")
	// ErrNoSuchIPPool .
	ErrNoSuchIPPool = errors.New("no such IPPool")
	// ErrTooLargeMaskBits .
	ErrTooLargeMaskBits = errors.New("too large mask bits")
	// ErrTooSmallMaskBits .
	ErrTooSmallMaskBits = errors.New("too small mask bits")
	// ErrInsufficientBlocks .
	ErrInsufficientBlocks = errors.New("insufficient blocks")
	// ErrInsufficientIP .
	ErrInsufficientIP = errors.New("insufficient free IP")
	// ErrIPIsnotAssigned .
	ErrIPIsnotAssigned = errors.New("IP isn't assigned")

	// ErrSerializedTaskAborted .
	ErrSerializedTaskAborted = errors.New("serialized task was aborted in advance")

	// ErrTimeout .
	ErrTimeout = errors.New("timed out")
	// ErrNotImplemented .
	ErrNotImplemented = errors.New("does not implemented")

	// ErrFolderExists .
	ErrFolderExists = errors.New("destination folder exists")
	// ErrDestinationInvalid .
	ErrDestinationInvalid = errors.New("destination path not valid")
	// ErrNotValidCopyStatus .
	ErrNotValidCopyStatus = errors.New("cannot copy in this status")
	// ErrNotValidLogStatus .
	ErrNotValidLogStatus = errors.New("cannot read log in this status")

	// ErrImageHubNotConfigured .
	ErrImageHubNotConfigured = errors.New("ImageHub is not set")
	// ErrImageFileNotExists .
	ErrImageFileNotExists = errors.New("Image File not exists")
	// ErrLoadImage
	ErrLoadImage = errors.New("Failed to load image")

	ErrFlockLocked = errors.New("flock locked")

	ErrUnknownNetworkDriver = errors.New("unknown network driver")
)
