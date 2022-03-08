package snapshot

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/utils"
	virtutil "github.com/projecteru2/yavirt/virt/util"
)

// Bot .
type Bot interface {
	Close() error
	Create(vol *model.Volume) error
	Commit(model.Snapshots) error
	Restore(vol *model.Volume, chain model.Snapshots) error
	Delete() error
	Upload(force bool) error
	Download(*model.Snapshot) error
	DeleteFromBackupStorage() error
}

type bot struct {
	snap  *Snapshot
	flock *util.Flock
}

func newVirtSnap(snap *Snapshot) (Bot, error) {
	var vs = &bot{
		snap: snap,
	}
	vs.flock = vs.newFlock()

	if err := vs.flock.Trylock(); err != nil {
		return nil, errors.Trace(err)
	}

	return vs, nil
}

// Create .
func (v *bot) Create(vol *model.Volume) error {
	tempFilepath := getTemporaryFilepath(vol.Filepath())

	if err := virtutil.CreateSnapshot(context.Background(), vol.Filepath(), tempFilepath); err != nil {
		return errors.Trace(err)
	}

	if err := sh.Copy(vol.Filepath(), v.snap.Filepath()); err != nil {
		return errors.Trace(err)
	}

	if err := virtutil.RebaseImage(context.Background(), tempFilepath, v.snap.Filepath()); err != nil {
		return errors.Trace(err)
	}

	return sh.Move(tempFilepath, vol.Filepath())
}

// Commit .
func (v *bot) Commit(chain model.Snapshots) error {

	if chain.Len() == 1 {
		return nil
	}

	for i := 0; i < chain.Len()-1; i++ {

		if err := virtutil.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
			return errors.Trace(err)
		}

		if err := sh.Remove(chain[i].Filepath()); err != nil {
			return errors.Trace(err)
		}
	}

	// Change name of the root snapshot to the current snapshot
	return sh.Move(chain[chain.Len()-1].Filepath(), chain[0].Filepath())
}

// Restore .
func (v *bot) Restore(vol *model.Volume, chain model.Snapshots) error {

	for i := 0; i < chain.Len(); i++ {
		if err := sh.Copy(chain[i].Filepath(), getTemporaryFilepath(chain[i].Filepath())); err != nil {
			return errors.Trace(err)
		}
	}

	for i := 0; i < chain.Len()-1; i++ {

		if err := virtutil.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
			return errors.Trace(err)
		}

		if err := sh.Remove(chain[i].Filepath()); err != nil {
			return errors.Trace(err)
		}
	}

	if err := sh.Move(chain[chain.Len()-1].Filepath(), vol.Filepath()); err != nil {
		return errors.Trace(err)
	}

	for i := 0; i < chain.Len(); i++ {
		if err := sh.Move(getTemporaryFilepath(chain[i].Filepath()), chain[i].Filepath()); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

// Delete .
func (v *bot) Delete() error {
	return sh.Remove(v.snap.Filepath())
}

// Upload .
func (v *bot) Upload(force bool) error {
	// TODO: implement
	// force = True means overwrite the file in backup storage
	return nil
}

// Download .
func (v *bot) Download(snapmod *model.Snapshot) error {
	// TODO: implement
	// Need to check whether file already exist
	return nil
}

// DeleteFromBackupStorage .
func (v *bot) DeleteFromBackupStorage() error {
	// TODO: implement
	return nil
}

func getTemporaryFilepath(filepath string) string {
	return filepath + ".temp"
}

func (v *bot) newFlock() *util.Flock {
	fn := fmt.Sprintf("%s.flock", v.snap.ID)
	fpth := filepath.Join(config.Conf.VirtFlockDir, fn)
	return util.NewFlock(fpth)
}

func (v *bot) Close() error {
	v.flock.Close()
	return nil
}
