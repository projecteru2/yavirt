package types

import "time"

const (
	DestroyOp         Operator = "destroy"
	DieOp             Operator = "die"
	StopOp            Operator = "stop"
	StartOp           Operator = "start"
	SuspendOp         Operator = "suspend"
	ResumeOp          Operator = "resume"
	CreateOp          Operator = "create"
	ExecuteOp         Operator = "execute"
	ResizeOp          Operator = "resize"
	ResetSysDiskOp    Operator = "reset-sys-disk"
	FSFreezeOP        Operator = "fs-freeze"
	FSThawOP          Operator = "fs-thaw"
	MiscOp            Operator = "misc"
	CreateSnapshotOp  Operator = "create-snapshot"
	CommitSnapshotOp  Operator = "commit-snapshot"
	RestoreSnapshotOp Operator = "restore-snapshot"
)

const (
	EventTypeGuest = "guest"
)

type Operator string

func (op Operator) String() string {
	return string(op)
}

type Event struct {
	ID   string
	Type string
	Op   Operator
	Time time.Time
}
