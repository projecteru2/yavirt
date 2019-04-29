package snapshot

import (
	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/model"
)

// Interface .
type Interface interface {
	Create(vol *model.Volume) error
	Commit(snaps model.Snapshots) (model.Snapshots, error)
	Delete() error
	Restore(vol *model.Volume, snaps model.Snapshots) error
	Upload(force bool) error
	Download(*model.Snapshot) error
}

// Snapshot .
type Snapshot struct {
	*model.Snapshot
	newBot func(*Snapshot) (Bot, error)
}

// New .
func New(mod *model.Snapshot) *Snapshot {
	return &Snapshot{
		Snapshot: mod,
		newBot:   newVirtSnap,
	}
}

// Delete meta and file in backup storage
func (snap *Snapshot) Delete() error {

	if err := snap.botOperate(func(bot Bot) error {
		return bot.DeleteFromBackupStorage()
	}); err != nil {
		return errors.Trace(err)
	}

	if err := snap.botOperate(func(bot Bot) error {
		return bot.Delete()
	}); err != nil {
		return errors.Trace(err)
	}

	if err := snap.Model().Delete(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Create external snapshot and return a list of Volume model represent volume that newly in use.
func (snap *Snapshot) Create(vol *model.Volume) error {
	if err := snap.botOperate(func(bot Bot) error {
		return bot.Create(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return snap.Upload(false)
}

// Commit current snapshot and snapshots before current snapshot to the root
// Root(Full snapshot) -> Snap1 -> Snap2 -> Snap3 -> Vol in-use
// After Snap2.Commit()
// NewRoot(Full snapshot with name same as Snap 2) -> Snap3 -> Vol in-use
// Return list of snapshot meta that needed to be removed
func (snap *Snapshot) Commit(snaps model.Snapshots) (model.Snapshots, error) {
	chain, err := snap.getChain(snaps)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err = snap.downloadSnapshots(chain); err != nil {
		return nil, errors.Trace(err)
	}

	if err = snap.botOperate(func(bot Bot) error {
		return bot.Commit(chain)
	}); err != nil {
		return nil, errors.Trace(err)
	}

	if err = snap.Upload(true); err != nil {
		return nil, errors.Trace(err)
	}

	return chain[1:], nil
}

// Restore .
func (snap *Snapshot) Restore(vol *model.Volume, snaps model.Snapshots) error {
	chain, err := snap.getChain(snaps)
	if err != nil {
		return errors.Trace(err)
	}

	if err := snap.downloadSnapshots(chain); err != nil {
		return errors.Trace(err)
	}

	if err := snap.botOperate(func(bot Bot) error {
		return bot.Restore(vol, chain)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Upload .
func (snap *Snapshot) Upload(force bool) error {
	if err := snap.botOperate(func(bot Bot) error {
		return bot.Upload(force)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Download .
func (snap *Snapshot) Download(snapmod *model.Snapshot) error {
	if err := snap.botOperate(func(bot Bot) error {
		return bot.Download(snapmod)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// // check whether the snapshot is the last snapshot on the chain
// // (not exist other snapshot use this snapshot as backing file)
// func (snap *Snapshot) checkSnapshotIsLatest(snaps model.Snapshots) bool {
// 	isLatest := true
// 	for _, s := range snaps {
// 		if s.BaseSnapshotID == snap.ID {
// 			isLatest = false
// 		}
// 	}
// 	return isLatest
// }

// calculate the whole chain
func (snap *Snapshot) getChain(snaps model.Snapshots) (model.Snapshots, error) {

	if _, err := snaps.Find(snap.ID); err != nil {
		return nil, errors.Trace(err)
	}

	snapIDMap := make(map[string]*model.Snapshot)
	for _, s := range snaps {
		snapIDMap[s.ID] = s
	}

	var chain model.Snapshots
	chain = append(chain, snap.Model())
	currentID := snap.BaseSnapshotID
	for len(currentID) > 0 {
		chain = append(chain, snapIDMap[currentID])
		currentID = snapIDMap[currentID].BaseSnapshotID
	}

	return chain, nil
}

// download list of snapshot files
func (snap *Snapshot) downloadSnapshots(snaps model.Snapshots) error {
	for _, s := range snaps {
		if err := snap.Download(s); err != nil {
			return err
		}
	}
	return nil
}

// Model .
func (snap *Snapshot) Model() *model.Snapshot {
	return snap.Snapshot
}

func (snap *Snapshot) botOperate(fn func(Bot) error) error {
	bot, err := snap.newBot(snap)
	if err != nil {
		return errors.Trace(err)
	}

	defer bot.Close()

	return fn(bot)
}
