package factory

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	fs = "ext4"
	// Disable backing up of the device/partition
	backupDump = 0
	// Enable fsck checking the device/partition for errors at boot time.
	fsckPass = 2
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
func Amplify(vol volume.Volume, delta int64, dom libvirt.Domain, ga agent.Interface, devPath string) (normDelta int64, err error) {
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
		err = interutils.AmplifyImage(context.Background(), vol.QemuImagePath(), delta)
	case libvirt.DomainRunning:
		err = amplifyOnline(newCap, dom, ga, devPath)
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

// Unmount .
func Unmount(vol volume.Volume, ga agent.Interface, devPath string) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout)
	defer cancel()

	log.WithFunc("volume.Umount").Debugf(ctx, "Umount: umount %s", devPath)
	cmds := []string{"umount", devPath}
	st := <-ga.ExecOutput(ctx, cmds[0], cmds[1:]...)
	if err := st.Error(); err != nil {
		log.WithFunc("volume.Umount").Warnf(ctx, "failed to run `%s`: %s", strings.Join(cmds, " "), err)
	}

	log.Debugf(ctx, "Umount: save fstab")
	escapeDir := strings.ReplaceAll(vol.GetMountDir(), "/", "\\/")
	regex := fmt.Sprintf("/%s/d", escapeDir)
	cmds = []string{"sed", "-i", regex, "/etc/fstab"}
	st = <-ga.ExecOutput(ctx, cmds[0], cmds[1:]...)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "failed to run `%v`", strings.Join(cmds, " "))
	}
	return nil

}

// Mount .
func Mount(vol volume.Volume, ga agent.Interface, devPath string) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout)
	defer cancel()

	log.Debugf(ctx, "Mount: format")
	if err := format(ctx, ga, vol, devPath); err != nil {
		return errors.Wrap(err, "")
	}

	log.Debugf(ctx, "Mount: mount")
	if err := mount(ctx, ga, vol, devPath); err != nil {
		return errors.Wrap(err, "")
	}

	log.Debugf(ctx, "Mount: save fstab")
	if err := saveFstab(ctx, ga, vol, devPath); err != nil {
		return errors.Wrap(err, "")
	}

	log.Debugf(ctx, "Mount: amplify if necessary")
	switch amplified, err := isAmplifying(ctx, ga, vol, devPath); {
	case err != nil:
		return errors.Wrap(err, "")

	case amplified:
		return amplifyDiskInGuest(ctx, ga, devPath)

	default:
		return nil
	}
}

func mount(ctx context.Context, ga agent.Interface, v volume.Volume, devPath string) error {
	var mnt = v.GetMountDir()
	var st = <-ga.Exec(ctx, "mkdir", "-p", mnt)
	if err := st.Error(); err != nil {
		return errors.Wrapf(err, "mkdir %s failed", mnt)
	}

	st = <-ga.ExecOutput(ctx, "mount", "-t", fs, devPath, mnt)
	_, _, err := st.CheckStdio(func(_, se []byte) bool {
		return bytes.Contains(se, []byte("already mounted"))
	})
	if err != nil {
		return errors.Wrapf(err, "mount %s failed", mnt)
	}
	return nil
}

func saveFstab(ctx context.Context, ga agent.Interface, v volume.Volume, devPath string) error {
	var blkid, err = ga.Blkid(ctx, devPath)
	if err != nil {
		return errors.Wrap(err, "")
	}

	switch exists, err := ga.Grep(ctx, blkid, types.FstabFile); {
	case err != nil:
		return errors.Wrap(err, "")
	case exists:
		return nil
	}

	var line = fmt.Sprintf("\nUUID=%s %s %s defaults %d %d",
		blkid, v.GetMountDir(), fs, backupDump, fsckPass)

	return ga.AppendLine(ctx, types.FstabFile, []byte(line))
}

func format(ctx context.Context, ga agent.Interface, v volume.Volume, devPath string) error {
	switch formatted, err := isFormatted(ctx, ga, v); {
	case err != nil:
		return errors.Wrap(err, "")
	case formatted:
		return nil
	}

	if err := fdisk(ctx, ga, devPath); err != nil {
		return errors.Wrap(err, "")
	}

	return ga.Touch(ctx, formattedFlagPath(v))
}

// parted -s /dev/vdN mklabel gpt
// parted -s /dev/vdN mkpart primary 1049K -- -1
// mkfs -F -t ext4 /dev/vdN
func fdisk(ctx context.Context, ga agent.Interface, devPath string) error {
	var cmds = [][]string{
		{"parted", "-s", devPath, "mklabel", "gpt"},
		{"parted", "-s", devPath, "mkpart", "primary", "1049K", "--", "-1"},
		{"mkfs", "-F", "-t", fs, devPath},
	}
	return base.ExecCommands(ctx, ga, cmds)
}

func isFormatted(ctx context.Context, ga agent.Interface, v volume.Volume) (bool, error) {
	return ga.IsFile(ctx, formattedFlagPath(v))
}

func formattedFlagPath(v volume.Volume) string {
	return fmt.Sprintf("/etc/%s", v.Name())
}

func isAmplifying(ctx context.Context, ga agent.Interface, v volume.Volume, devPath string) (bool, error) {
	mbs, err := getMountedBlocks(ctx, ga, v)
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	cap, err := agent.NewParted(ga, devPath).GetSize(ctx) //nolint
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	mbs = int64(float64(mbs) * (1 + configs.Conf.ResizeVolumeMinRatio))
	cap >>= 10 //nolint // in bytes, aka. 1K-blocks.

	return cap > mbs, nil
}

func getMountedBlocks(ctx context.Context, ga agent.Interface, v volume.Volume) (int64, error) {
	df, err := ga.GetDiskfree(ctx, v.GetMountDir())
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return df.Blocks, nil
}

func amplifyOnline(newCap int64, dom libvirt.Domain, ga agent.Interface, devPath string) error {
	devname := filepath.Base(devPath)
	if err := dom.AmplifyVolume(devname, uint64(newCap)); err != nil {
		return errors.Wrap(err, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout)
	defer cancel()
	return amplifyDiskInGuest(ctx, ga, devPath)
}

func amplifyDiskInGuest(ctx context.Context, ga agent.Interface, devPath string) error {
	// NOTICE:
	//   Actually, volume raw devices aren't necessary for re-parting.

	stoppedServices, err := base.StopSystemdServices(ctx, ga, devPath)
	if err != nil {
		return errors.Wrap(err, "")
	}

	cmds := [][]string{
		{"umount", devPath},
		{"partprobe"},
		{"e2fsck", "-fy", devPath},
		{"resize2fs", devPath},
		{"mount", "-a"},
	}

	if err := base.ExecCommands(ctx, ga, cmds); err != nil {
		return errors.Wrap(err, "")
	}

	if err := base.RestartSystemdServices(ctx, ga, stoppedServices); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}
