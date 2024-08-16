package factory

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Undefine .
func Undefine(vol volume.Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	if err := api.DeleteAll(); err != nil {
		return errors.Wrap(err, "")
	}
	if err := vol.Cleanup(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func Create(vol volume.Volume) (func(), error) {
	if err := volume.WithLocker(vol, func() error {
		return vol.PrepareDataDisk(context.TODO())
	}); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if err := vol.Save(); err != nil {
		if ue := Undefine(vol); ue != nil {
			err = errors.CombineErrors(err, ue)
		}
		return nil, errors.Wrap(err, "")
	}

	rb := func() {
		if err := Undefine(vol); err != nil {
			log.WithFunc("volume.Create").Error(context.TODO(), err)
			return
		}
		if err := vol.Delete(true); err != nil {
			log.WithFunc("volume.Create").Error(context.TODO(), err)
		}
	}

	return rb, nil
}

// Amplify .
func Amplify(vol volume.Volume, delta int64, dom libvirt.Domain, ga agent.Interface) (normDelta int64, err error) {
	if err := vol.Lock(); err != nil {
		return 0, errors.Wrap(err, "")
	}
	defer vol.Unlock()

	normDelta = utils.NormalizeMultiple1024(delta)
	sizeInBytes := vol.GetSize()
	newCap := sizeInBytes + normDelta
	if newCap > configs.Conf.Resource.MaxVolumeCap {
		return 0, errors.Wrapf(terrors.ErrInvalidValue, "exceeds the max cap: %d", configs.Conf.Resource.MaxVolumeCap)
	}

	least := utils.Max(
		configs.Conf.ResizeVolumeMinSize,
		int64(float64(sizeInBytes)*configs.Conf.ResizeVolumeMinRatio),
	)
	if least > normDelta {
		return 0, errors.Wrapf(terrors.ErrInvalidValue, "invalid cap: at least %d, but %d",
			sizeInBytes+least, sizeInBytes+normDelta)
	}

	st, err := dom.GetState()
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	switch st {
	case libvirt.DomainShutoff:
		err = vol.AmplifyOffline(context.Background(), delta)
	case libvirt.DomainRunning:
		err = vol.AmplifyOnline(newCap, dom, ga)
	default:
		err = types.NewDomainStatesErr(st, libvirt.DomainShutoff, libvirt.DomainRunning)
	}
	if err != nil {
		return 0, err
	}

	vol.SetSize(newCap)
	return normDelta, vol.Save()
}

// Check .
func Check(vol volume.Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	return vol.Check()
}

// Repair .
func Repair(vol volume.Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	return vol.Repair()
}

// CreateSnapshot .
func CreateSnapshot(vol volume.Volume) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Create()
}

// CommitSnapshot .
func CommitSnapshot(vol volume.Volume, snapID string) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Commit(snapID)
}

// CommitSnapshotByDay Commit snapshots created `day` days ago.
func CommitSnapshotByDay(vol volume.Volume, day int) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.CommitByDay(day)
}

// RestoreSnapshot .
func RestoreSnapshot(vol volume.Volume, snapID string) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	api := vol.NewSnapshotAPI()
	return api.Restore(snapID)
}

// for ceph, before create snapshot, we need run fsfreeze
func FSFreeze(ctx context.Context, ga agent.Interface, v volume.Volume, unfreeze bool) error {
	var cmd []string
	if unfreeze {
		cmd = []string{"fsfreeze", "--unfreeze", v.GetMountDir()}
	} else {
		cmd = []string{"fsfreeze", "--freeze", v.GetMountDir()}
	}
	var st = <-ga.ExecOutput(ctx, cmd[0], cmd[1:]...)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "%v", cmd)
	}
	return nil
}

// Umount .
func Umount(vol volume.Volume, ga agent.Interface) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout)
	defer cancel()
	return vol.Umount(ctx, ga)
}

// Mount .
func Mount(vol volume.Volume, ga agent.Interface) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout)
	defer cancel()
	return vol.Mount(ctx, ga)
}
