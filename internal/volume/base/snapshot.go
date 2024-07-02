package base

import "github.com/projecteru2/yavirt/internal/meta"

// SnapshotAPI .
type SnapshotAPI interface {
	List() Snapshots
	Create() error
	Commit(rootID string) error
	CommitByDay(day int) error
	Delete(id string) error
	DeleteAll() error
	Restore(rootID string) error
	Upload(id string, force bool) error
	Download(id string) error
}

type Snapshot interface {
	meta.GenericInterface
}

type Snapshots []Snapshot
