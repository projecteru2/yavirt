package snapshot

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	virtutils "github.com/projecteru2/yavirt/internal/virt/utils"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Bot .
type Bot interface {
	Close() error
	Create(vol *models.Volume) error
	Commit(models.Snapshots) error
	Restore(vol *models.Volume, chain models.Snapshots) error
	Delete() error
	Upload(force bool) error
	Download(*models.Snapshot) error
	DeleteFromBackupStorage() error
}

type bot struct {
	snap  *Snapshot
	flock *utils.Flock
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
func (v *bot) Create(vol *models.Volume) error {
	tempFilepath := getTemporaryFilepath(vol.Filepath())

	if err := virtutils.CreateSnapshot(context.Background(), vol.Filepath(), tempFilepath); err != nil {
		return errors.Trace(err)
	}

	if err := sh.Copy(vol.Filepath(), v.snap.Filepath()); err != nil {
		return errors.Trace(err)
	}

	if err := virtutils.RebaseImage(context.Background(), tempFilepath, v.snap.Filepath()); err != nil {
		return errors.Trace(err)
	}

	return sh.Move(tempFilepath, vol.Filepath())
}

// Commit .
func (v *bot) Commit(chain models.Snapshots) error {

	if chain.Len() == 1 {
		return nil
	}

	for i := 0; i < chain.Len()-1; i++ {

		if err := virtutils.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
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
func (v *bot) Restore(vol *models.Volume, chain models.Snapshots) error {

	for i := 0; i < chain.Len(); i++ {
		if err := sh.Copy(chain[i].Filepath(), getTemporaryFilepath(chain[i].Filepath())); err != nil {
			return errors.Trace(err)
		}
	}

	for i := 0; i < chain.Len()-1; i++ {

		if err := virtutils.CommitImage(context.Background(), chain[i].Filepath()); err != nil {
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
func (v *bot) Download(snapmod *models.Snapshot) error {
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

func (v *bot) newFlock() *utils.Flock {
	fn := fmt.Sprintf("%s.flock", v.snap.ID)
	fpth := filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

func (v *bot) Close() error {
	v.flock.Close()
	return nil
}
