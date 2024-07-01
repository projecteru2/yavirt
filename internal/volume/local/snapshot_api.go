package local

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	virtutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/sh"
)

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
	ans := base.Snapshots{}
	for _, snap := range api.vol.Snaps {
		ans = append(ans, snap)
	}
	return ans
}

// Delete meta and file in backup storage
func (api *SnapshotAPI) Delete(id string) error {
	if err := api.delete(id); err != nil {
		return errors.Wrap(err, "")
	}
	return api.vol.Save()
}

func (api *SnapshotAPI) delete(id string) error {
	vol := api.vol
	snap, err := LoadSnapshot(id)
	if err != nil {
		return errors.Wrap(err, "")
	}
	// TODO delete backups in backup storage
	if err := sh.Remove(snap.Filepath()); err != nil {
		return errors.Wrap(err, "")
	}

	vol.RemoveSnap(snap.ID)
	if err := snap.Delete(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (api *SnapshotAPI) DeleteAll() error {
	for _, id := range api.vol.SnapIDs {
		if err := api.delete(id); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return api.vol.Save()
}

// Create external snapshot and return a list of Volume model represent volume that newly in use.
func (api *SnapshotAPI) Create() error {
	vol := api.vol
	snapmod := NewSnapShot(vol.ID)
	snapmod.GenerateID()

	// change the qcow2 files
	volFname := api.vol.Filepath()
	tempFilepath := getTemporaryFilepath(api.vol.Filepath())

	if err := virtutils.CreateSnapshot(context.Background(), volFname, tempFilepath); err != nil {
		return errors.Wrap(err, "")
	}

	if err := sh.Copy(volFname, snapmod.Filepath()); err != nil {
		return errors.Wrap(err, "")
	}

	if err := virtutils.RebaseImage(context.Background(), tempFilepath, snapmod.Filepath()); err != nil {
		return errors.Wrap(err, "")
	}

	if err := sh.Move(tempFilepath, volFname); err != nil {
		return errors.Wrap(err, "")
	}
	if err := api.upload(snapmod, false); err != nil {
		return errors.Wrap(err, "")
	}

	if err := vol.AppendSnaps(snapmod); err != nil {
		return errors.Wrap(err, "")
	}

	// save snapshot and volume to data store
	snapmod.BaseSnapshotID = vol.BaseSnapshotID
	vol.BaseSnapshotID = snapmod.ID
	if err := snapmod.Create(); err != nil {
		return errors.Wrap(err, "")
	}
	return api.vol.Save()
}

// Commit current snapshot and snapshots before current snapshot to the root
// Root(Full snapshot) -> Snap1 -> Snap2 -> Snap3 -> Vol in-use
// After Snap2.Commit()
// NewRoot(Full snapshot with name same as Snap 2) -> Snap3 -> Vol in-use
func (api *SnapshotAPI) Commit(rootID string) error {
	snap, err := LoadSnapshot(rootID)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return api.commit(snap)
}

func (api *SnapshotAPI) commit(snap *Snapshot) error {
	vol := api.vol
	snaps := vol.Snaps
	chain, err := getChain(snap, snaps)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if chain.Len() <= 1 {
		return nil
	}

	if err = api.downloadSnapshots(chain); err != nil {
		return errors.Wrap(err, "")
	}

	for i := 0; i < chain.Len()-1; i++ {
		if err := virtutils.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
			return errors.Wrap(err, "")
		}
		if err := sh.Remove(chain[i].Filepath()); err != nil {
			return errors.Wrap(err, "")
		}
	}
	// Change name of the root snapshot to the current snapshot
	if err := sh.Move(chain[chain.Len()-1].Filepath(), chain[0].Filepath()); err != nil {
		return errors.Wrap(err, "")
	}
	if err = api.upload(snap, true); err != nil {
		return errors.Wrap(err, "")
	}

	// delete snapshot meta data
	for _, snap := range chain[1:] {
		api.vol.RemoveSnap(snap.ID)
		if err := snap.Delete(); err != nil {
			return errors.Wrap(err, "")
		}
	}
	if err := api.vol.Save(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (api *SnapshotAPI) CommitByDay(day int) error {
	vol := api.vol
	date := time.Now().AddDate(0, 0, -day).Unix()
	targetIdx := -1
	for i, s := range vol.Snaps {
		if s.CreatedTime > date { // Find the first snapshot that is created later than x days before
			targetIdx = i
		}
	}

	// No need to commit if all snapshot created within x days
	if targetIdx == 0 {
		return nil
	}

	// If all snapshot create x days before, keep the last one
	if targetIdx == -1 {
		targetIdx = vol.Snaps.Len() - 1
	}

	return api.commit(vol.Snaps[targetIdx])
}

// Restore .
func (api *SnapshotAPI) Restore(rootID string) error {
	vol := api.vol
	snap, err := LoadSnapshot(rootID)
	if err != nil {
		return errors.Wrap(err, "")
	}
	snaps := vol.Snaps
	chain, err := getChain(snap, snaps)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := api.downloadSnapshots(chain); err != nil {
		return errors.Wrap(err, "")
	}

	for i := 0; i < chain.Len(); i++ {
		if err := sh.Copy(chain[i].Filepath(), getTemporaryFilepath(chain[i].Filepath())); err != nil {
			return errors.Wrap(err, "")
		}
	}

	for i := 0; i < chain.Len()-1; i++ {
		if err := virtutils.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
			return errors.Wrap(err, "")
		}

		if err := sh.Remove(chain[i].Filepath()); err != nil {
			return errors.Wrap(err, "")
		}
	}

	if err := sh.Move(chain[chain.Len()-1].Filepath(), vol.Filepath()); err != nil {
		return errors.Wrap(err, "")
	}

	for i := 0; i < chain.Len(); i++ {
		if err := sh.Move(getTemporaryFilepath(chain[i].Filepath()), chain[i].Filepath()); err != nil {
			return errors.Wrap(err, "")
		}
	}
	vol.BaseSnapshotID = rootID
	return vol.Save()
}

// Upload .
func (api *SnapshotAPI) Upload(id string, force bool) error {
	snapmod, err := LoadSnapshot(id)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return api.upload(snapmod, force)
}

func (api *SnapshotAPI) upload(snapmod *Snapshot, force bool) error { //nolint
	// TODO upload to backup storage

	return nil
}

// Download .
func (api *SnapshotAPI) Download(id string) error {
	snapmod, err := LoadSnapshot(id)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return api.download(snapmod)
}

func (api *SnapshotAPI) download(snapmod *Snapshot) error { //nolint
	// TODO download from backup storage
	return nil
}

// // check whether the snapshot is the last snapshot on the chain
// // (not exist other snapshot use this snapshot as backing file)
// func (snap *Snapshot) checkSnapshotIsLatest(snaps Snapshots) bool {
// 	isLatest := true
// 	for _, s := range snaps {
// 		if s.BaseSnapshotID == snap.ID {
// 			isLatest = false
// 		}
// 	}
// 	return isLatest
// }

// calculate the whole chain
func getChain(rootSnap *Snapshot, snaps Snapshots) (Snapshots, error) {

	if _, err := snaps.Find(rootSnap.ID); err != nil {
		return nil, errors.Wrap(err, "")
	}

	snapIDMap := make(map[string]*Snapshot)
	for _, s := range snaps {
		snapIDMap[s.ID] = s
	}

	var chain Snapshots
	chain = append(chain, rootSnap)
	currentID := rootSnap.BaseSnapshotID
	for len(currentID) > 0 {
		chain = append(chain, snapIDMap[currentID])
		currentID = snapIDMap[currentID].BaseSnapshotID
	}

	return chain, nil
}

// download list of snapshot files
func (api *SnapshotAPI) downloadSnapshots(snaps Snapshots) error {
	for _, s := range snaps {
		if err := api.download(s); err != nil {
			return err
		}
	}
	return nil
}

func getTemporaryFilepath(filepath string) string {
	return filepath + ".temp"
}
