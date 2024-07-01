package rbd

import "github.com/projecteru2/yavirt/internal/volume/base"

// SnapshotAPI .
type SnapshotAPI struct {
	vol *Volume
}

// New .
func newSnapshotAPI(v *Volume) *SnapshotAPI {
	return &SnapshotAPI{
		vol: v,
	}
}

func (api *SnapshotAPI) List() base.Snapshots {
	return nil
}
func (api *SnapshotAPI) Create() error {
	return nil
}
func (api *SnapshotAPI) Commit(rootID string) error { //nolint
	return nil
}
func (api *SnapshotAPI) CommitByDay(day int) error { //nolint
	return nil
}
func (api *SnapshotAPI) Delete(id string) error { //nolint
	return nil
}
func (api *SnapshotAPI) DeleteAll() error {
	return nil
}
func (api *SnapshotAPI) Restore(rootID string) error { //nolint
	return nil
}
func (api *SnapshotAPI) Upload(id string, force bool) error { //nolint
	return nil
}
func (api *SnapshotAPI) Download(id string) error { //nolint
	return nil
}
