package meta

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
	// StatusFreeze .
	StatusFreeze = "frozen"
	// StatusThaw .
	StatusThaw = "thawed"
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
