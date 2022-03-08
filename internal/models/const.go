package models

const (
	// StatusPending .
	StatusPending = "pending"
	// StatusCreating .
	StatusCreating = "creating"
	// StatusStarting .
	StatusStarting = "starting"
	// StatusRunning .
	StatusRunning = "running"
	// StatusStopping .
	StatusStopping = "stopping"
	// StatusStopped .
	StatusStopped = "stopped"
	// StatusMigrating .
	StatusMigrating = "migrating"
	// StatusResizing .
	StatusResizing = "resizing"
	// StatusCapturing .
	StatusCapturing = "capturing"
	// StatusCaptured .
	StatusCaptured = "captured"
	// StatusDestroying .
	StatusDestroying = "destroying"
	// StatusDestroyed .
	StatusDestroyed = "destroyed"
	// StatusPausing .
	StatusPausing = "pausing"
	// StatusPaused .
	StatusPaused = "paused"
	// StatusResuming .
	StatusResuming = "resuming"

	// VolDataType .
	VolDataType = "dat"
	// VolSysType .
	VolSysType = "sys"
	// VolQcow2Format .
	VolQcow2Format = "qcow2"

	// SnapshotFullType .
	SnapshotFullType = "full"
	// SnapshotIncrementalType .
	SnapshotIncrementalType = "incremental"

	// HostVirtType .
	HostVirtType = "virt"
	// HostMetaType .
	HostMetaType = "meta"

	// DistroUbuntu .
	DistroUbuntu = "ubuntu"
	// DistroCentOS .
	DistroCentOS = "centos"

	// UserImagePrefix .
	UserImagePrefix = "uimg"
	// ImageSys indicates the image is a system version.
	ImageSys = "sys"
	// ImageUser indicates the image was captured by user.
	ImageUser = "user"

	// LabelPublish .
	LabelPublish = "Publish"
	// LabelHealthCheck .
	LabelHealthCheck = "HealthCheck"

	// MaxMaskBits at most /30
	MaxMaskBits = 30
	// MinMaskBits at least /1
	MinMaskBits = 1
	// MaxMaskBitsForBlocks at most /23
	MaxMaskBitsForBlocks = 23
	// BlockFlagSpawned .
	BlockFlagSpawned = 1
	// BlockFlagUnspawned .
	BlockFlagUnspawned = 0
	// IPFlagFree .
	IPFlagFree = 0
	// IPFlagAssigned .
	IPFlagAssigned = 1
	// MaxBlockIPCount .
	MaxBlockIPCount = 256
)

// AllStatuses .
var AllStatuses = []string{
	StatusPending,
	StatusCreating,
	StatusStarting,
	StatusRunning,
	StatusStopping,
	StatusStopped,
	StatusCapturing,
	StatusCaptured,
	StatusMigrating,
	StatusResizing,
	StatusDestroying,
	StatusDestroyed,
}
