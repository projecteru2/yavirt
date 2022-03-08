package volume

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/domain"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	gfsmocks "github.com/projecteru2/yavirt/internal/virt/guestfs/mocks"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/internal/virt/snapshot"
	"github.com/projecteru2/yavirt/internal/virt/types"
	virtutils "github.com/projecteru2/yavirt/internal/virt/utils"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	fs = "ext4"
	// Disable backing up of the device/partition
	backupDump = 0
	// Enable fsck checking the device/partition for errors at boot time.
	fsckPass = 2
)

// Bot .
type Bot interface {
	Close() error
	Undefine() error
	Alloc() error
	AllocFromImage(models.Image) error
	Mount(ga agent.Interface, devPath string) error
	Amplify(delta int64, dom domain.Domain, ga agent.Interface, devPath string) error
	ConvertUserImage(user, name string) (*models.UserImage, error)
	Check() error
	Repair() error
	CreateSnapshot(*models.Snapshot) error
	CommitSnapshot(*models.Snapshot) error
	RestoreSnapshot(*models.Snapshot) error
	DeleteSnapshot(*models.Snapshot) error
	DeleteAllSnapshots() error
}

type bot struct {
	vol         *Volume
	flock       *utils.Flock
	newGuestfs  func(string) (guestfs.Guestfs, error)
	newSnapshot func(*models.Snapshot) snapshot.Interface
}

func newVirtVol(vol *Volume) (Bot, error) {
	var virt = &bot{
		vol:         vol,
		newGuestfs:  gfsx.New,
		newSnapshot: newSnapshot,
	}
	virt.flock = virt.newFlock()

	if err := virt.flock.Trylock(); err != nil {
		return nil, errors.Trace(err)
	}

	return virt, nil
}

func newSnapshot(snapmod *models.Snapshot) snapshot.Interface {
	return snapshot.New(snapmod)
}

func (v *bot) Undefine() error {
	return sh.Remove(v.vol.Filepath())
}

func (v *bot) Close() error {
	v.flock.Close()
	return nil
}

func (v *bot) DeleteAllSnapshots() error {
	for i := v.vol.Snaps.Len() - 1; i >= 0; i-- {
		if err := v.DeleteSnapshot(v.vol.Snaps[i]); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (v *bot) DeleteSnapshot(snapmod *models.Snapshot) error {
	fmt.Println("Destroying snap " + snapmod.ID)
	v.vol.RemoveSnap(snapmod.ID)
	if err := v.newSnapshot(snapmod).Delete(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *bot) CreateSnapshot(snapmod *models.Snapshot) error {
	snap := v.newSnapshot(snapmod)
	if err := snap.Create(v.vol.Model()); err != nil {
		return err
	}

	snapmod.BaseSnapshotID = v.vol.BaseSnapshotID
	v.vol.BaseSnapshotID = snapmod.ID

	return snapmod.Create()
}

func (v *bot) CommitSnapshot(snapmod *models.Snapshot) error {
	snap := v.newSnapshot(snapmod)
	snapsToRemoved, err := snap.Commit(v.vol.Snaps)
	if err != nil {
		return errors.Trace(err)
	}

	for _, s := range snapsToRemoved {
		if err := v.DeleteSnapshot(s); err != nil {
			return errors.Trace(err)
		}
	}

	snapmod.BaseSnapshotID = ""

	return snapmod.Save()
}

func (v *bot) RestoreSnapshot(snapmod *models.Snapshot) error {
	snap := v.newSnapshot(snapmod)
	err := snap.Restore(v.vol.Model(), v.vol.Snaps)
	if err != nil {
		return errors.Trace(err)
	}

	v.vol.BaseSnapshotID = snapmod.ID

	return nil
}

func (v *bot) Amplify(delta int64, dom domain.Domain, ga agent.Interface, devPath string) error {
	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Trace(err)

	case st == libvirt.DomainShutoff:
		return virtutils.AmplifyImage(context.Background(), v.vol.Filepath(), delta)

	case st == libvirt.DomainRunning:
		return v.amplifyOnline(delta, dom, ga, devPath)

	default:
		return types.NewDomainStatesErr(st, libvirt.DomainShutoff, libvirt.DomainRunning)
	}
}

func (v *bot) amplifyOnline(delta int64, dom domain.Domain, ga agent.Interface, devPath string) error {
	if err := dom.AmplifyVolume(v.vol.Filepath(), uint64(v.vol.Capacity+delta)); err != nil {
		return errors.Trace(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout.Duration())
	defer cancel()
	return v.amplify(ctx, ga, devPath)
}

func (v *bot) newFlock() *utils.Flock {
	var fn = fmt.Sprintf("%s.flock", v.vol.Name())
	var fpth = filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

func (v *bot) Mount(ga agent.Interface, devPath string) error {
	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GADiskTimeout.Duration())
	defer cancel()

	if err := v.format(ctx, ga, devPath); err != nil {
		return errors.Trace(err)
	}

	if err := v.mount(ctx, ga, devPath); err != nil {
		return errors.Trace(err)
	}

	if err := v.saveFstab(ctx, ga, devPath); err != nil {
		return errors.Trace(err)
	}

	switch amplified, err := v.isAmplifying(ctx, ga, devPath); {
	case err != nil:
		return errors.Trace(err)

	case amplified:
		return v.amplify(ctx, ga, devPath)

	default:
		return nil
	}
}

func (v *bot) isAmplifying(ctx context.Context, ga agent.Interface, devPath string) (bool, error) {
	mbs, err := v.getMountedBlocks(ctx, ga)
	if err != nil {
		return false, errors.Trace(err)
	}

	cap, err := agent.NewParted(ga, devPath).GetSize(ctx)
	if err != nil {
		return false, errors.Trace(err)
	}

	mbs = int64(float64(mbs) * (1 + configs.Conf.ResizeVolumeMinRatio))
	cap >>= 10 // in bytes, aka. 1K-blocks.

	return cap > mbs, nil
}

func (v *bot) getMountedBlocks(ctx context.Context, ga agent.Interface) (int64, error) {
	df, err := ga.GetDiskfree(ctx, v.vol.MountDir)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return df.Blocks, nil
}

func (v *bot) amplify(ctx context.Context, ga agent.Interface, devPath string) error {
	// NOTICE:
	//   Actually, volume raw devices aren't necessary for re-parting.

	stoppedServices, err := v.stopSystemdServices(ctx, ga, devPath)
	if err != nil {
		return errors.Trace(err)
	}

	cmds := [][]string{
		{"umount", v.vol.MountDir},
		{"partprobe"},
		{"e2fsck", "-fy", devPath},
		{"resize2fs", devPath},
		{"mount", "-a"},
	}

	if err := v.execCommands(ctx, ga, cmds); err != nil {
		return errors.Trace(err)
	}

	if err := v.restartSystemdServices(ctx, ga, stoppedServices); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (v *bot) saveFstab(ctx context.Context, ga agent.Interface, devPath string) error {
	var blkid, err = ga.Blkid(ctx, devPath)
	if err != nil {
		return errors.Trace(err)
	}

	switch exists, err := ga.Grep(ctx, blkid, types.FstabFile); {
	case err != nil:
		return errors.Trace(err)
	case exists:
		return nil
	}

	var line = fmt.Sprintf("\nUUID=%s %s %s defaults %d %d",
		blkid, v.vol.MountDir, fs, backupDump, fsckPass)

	return ga.AppendLine(types.FstabFile, []byte(line))
}

func (v *bot) mount(ctx context.Context, ga agent.Interface, devPath string) error {
	var mnt = v.vol.MountDir
	var st = <-ga.Exec(ctx, "mkdir", "-p", mnt)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, "mkdir %s failed", mnt)
	}

	st = <-ga.ExecOutput(ctx, "mount", "-t", fs, devPath, mnt)
	_, _, err := st.CheckStdio(func(_, se []byte) bool {
		return bytes.Contains(se, []byte("already mounted"))
	})
	return errors.Annotatef(err, "mount %s failed", mnt)
}

func (v *bot) format(ctx context.Context, ga agent.Interface, devPath string) error {
	switch formatted, err := v.isFormatted(ctx, ga); {
	case err != nil:
		return errors.Trace(err)
	case formatted:
		return nil
	}

	if err := v.fdisk(ctx, ga, devPath); err != nil {
		return errors.Trace(err)
	}

	return ga.Touch(ctx, v.formattedFlagPath())
}

func (v *bot) fdisk(ctx context.Context, ga agent.Interface, devPath string) error {
	var cmds = [][]string{
		{"parted", "-s", devPath, "mklabel", "gpt"},
		{"parted", "-s", devPath, "mkpart", "primary", "1049K", "--", "-1"},
		{"mkfs", "-F", "-t", fs, devPath},
	}
	return v.execCommands(ctx, ga, cmds)
}

func (v *bot) isFormatted(ctx context.Context, ga agent.Interface) (bool, error) {
	return ga.IsFile(ctx, v.formattedFlagPath())
}

func (v *bot) formattedFlagPath() string {
	return fmt.Sprintf("/etc/%s", v.vol.Name())
}

func (v *bot) execCommands(ctx context.Context, ga agent.Interface, cmds [][]string) error {
	for _, args := range cmds {
		var st = <-ga.ExecOutput(ctx, args[0], args[1:]...)
		if err := st.Error(); err != nil {
			return errors.Annotatef(err, "%v", args)
		}
	}
	return nil
}

func (v *bot) Alloc() error {
	var path = v.vol.Filepath()
	return virtutils.CreateImage(context.Background(), models.VolQcow2Format, path, v.vol.Capacity)
}

func (v *bot) AllocFromImage(img models.Image) error {
	if err := sh.Copy(img.Filepath(), v.vol.Filepath()); err != nil {
		return errors.Trace(err)
	}

	// Removes the image file form localhost if it's a user image.
	if img.GetType() == models.ImageUser {
		if err := sh.Remove(img.Filepath()); err != nil {
			// Prints the error message but it doesn't break the func.
			log.WarnStack(err)
		}
	}

	return nil
}

func (v *bot) ConvertUserImage(user, name string) (uimg *models.UserImage, err error) {
	uimg = models.NewUserImage(user, name, v.vol.Capacity)
	uimg.Distro = types.Unknown

	orig := uimg.Filepath()
	if err := sh.Copy(v.vol.Filepath(), orig); err != nil {
		return nil, errors.Trace(err)
	}
	defer func() {
		if err != nil {
			if re := sh.Remove(orig); re != nil {
				err = errors.Wrap(err, re)
			}
		}
	}()

	var gfs guestfs.Guestfs
	if gfs, err = v.newGuestfs(orig); err != nil {
		return nil, errors.Trace(err)
	}
	defer gfs.Close()

	if uimg.Distro, err = gfs.Distro(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = v.resetUserImage(gfs, uimg.Distro); err != nil {
		return nil, errors.Trace(err)
	}

	if err := uimg.NextVersion(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = sh.Move(orig, uimg.Filepath()); err != nil {
		return nil, errors.Trace(err)
	}

	if err = v.cleanObsoleteUserImages(uimg); err != nil {
		return nil, errors.Trace(err)
	}

	return uimg, nil
}

// Check .
func (v *bot) Check() error {
	return virtutils.Check(context.Background(), v.vol.Filepath())
}

// Repair .
func (v *bot) Repair() error {
	return virtutils.Repair(context.Background(), v.vol.Filepath())
}

func (v *bot) cleanObsoleteUserImages(uimg *models.UserImage) error {
	// TODO
	return nil
}

func (v *bot) resetUserImage(gfs guestfs.Guestfs, distro string) error {
	if err := v.resetFstab(gfs); err != nil {
		return errors.Trace(err)
	}

	return v.resetEth0(gfs, distro)
}

func (v *bot) resetFstab(gfs guestfs.Guestfs) error {
	origFstabEntries, err := gfs.GetFstabEntries()
	if err != nil {
		return errors.Trace(err)
	}

	blkids, err := gfs.GetBlkids()
	if err != nil {
		return errors.Trace(err)
	}

	var cont string
	for dev, entry := range origFstabEntries {
		if blkids.Exists(dev) {
			cont += fmt.Sprintf("%s\n", strings.TrimSpace(entry))
		}
	}

	return gfs.Write(types.FstabFile, cont)
}

func (v *bot) resetEth0(gfs guestfs.Guestfs, distro string) error {
	path, err := nic.GetEthFile(distro, "eth0")
	if err != nil {
		return errors.Trace(err)
	}
	return gfs.Remove(path)
}

func (v *bot) stopSystemdServices(ctx context.Context, ga agent.Interface, devPath string) ([]string, error) {
	var st = <-ga.ExecOutput(ctx, "fuser", "-m", devPath)
	so, se, err := st.Stdio()
	if err != nil && (len(so) > 0 || len(se) > 0) { // Fuser return status code 1 if no process running
		return nil, errors.Annotatef(err, "fuser on %s failed", devPath)
	}

	re := regexp.MustCompile(`[0-9]+`)
	pids := re.FindAllString(string(so), -1)

	var stoppedServices []string
	for _, pid := range pids {
		switch serviceName, err := v.findService(ctx, ga, pid); {
		case err != nil:
			return nil, errors.Trace(err)

		case len(serviceName) > 0:
			if err := v.stopSystemdService(ctx, ga, serviceName); err != nil {
				return nil, errors.Trace(err)
			}
			stoppedServices = append(stoppedServices, serviceName)

		default:
			continue
		}
	}

	return stoppedServices, nil
}

func (v *bot) stopSystemdService(ctx context.Context, ga agent.Interface, serviceName string) error {
	var st = <-ga.Exec(ctx, "systemctl", "stop", serviceName)
	if err := st.Error(); err != nil {
		return errors.Annotatef(err, "systemctl stop %s failed", serviceName)
	}

	return nil
}

func (v *bot) restartSystemdServices(ctx context.Context, ga agent.Interface, stoppedServices []string) error {
	for _, serviceName := range stoppedServices {
		var st = <-ga.Exec(ctx, "systemctl", "start", serviceName)
		if err := st.Error(); err != nil {
			return errors.Annotatef(err, "systemctl start %s failed", serviceName)
		}
	}
	return nil
}

func (v *bot) findService(ctx context.Context, ga agent.Interface, pid string) (string, error) {
	for {
		switch name, se := v.getServiceNameByPid(ctx, ga, pid); {
		case strings.HasPrefix(se, "Failed "): // Doesn't exist systemd unit with this pid
			ppid, err := v.getPpid(ctx, ga, pid)
			if err != nil {
				return "", errors.Trace(err)
			}
			pid = ppid

		case len(name) > 0:
			switch valid, err := v.isService(ctx, ga, name); {
			case err != nil:
				return "", errors.Trace(err)

			case valid:
				return name, nil

			default: // unit with this pid exist but not service type
				return "", nil
			}

		default:
			return "", nil
		}
	}
}

func (v *bot) getPpid(ctx context.Context, ga agent.Interface, pid string) (string, error) {
	var st = <-ga.ExecOutput(ctx, "ps", "--ppid", pid)
	so, _, err := st.Stdio()
	if err != nil {
		return "", errors.Annotatef(err, "find ppid for %s failed", pid)
	}
	if len(so) == 0 {
		return "", errors.Annotatef(err, "ppid for %s is empty", pid)
	}
	return string(so), nil
}

func (v *bot) getServiceNameByPid(ctx context.Context, ga agent.Interface, pid string) (string, string) {
	var st = <-ga.ExecOutput(ctx, "systemctl", "status", pid)
	so, se, err := st.Stdio()
	if err != nil {
		return "", string(se) // No service with this pid or service is stopped/failed
	}
	soSplit := strings.Fields(string(so))
	if len(soSplit) < 2 || len(soSplit[1]) == 0 {
		return "", ""
	}
	return soSplit[1], string(se)
}

func (v *bot) isService(ctx context.Context, ga agent.Interface, unitName string) (bool, error) {
	var st = <-ga.ExecOutput(ctx, "systemctl", "list-units", "--all", "-t", "service",
		"--full", "--no-legend", unitName)

	so, se, err := st.Stdio()
	if err != nil {
		if len(so) > 0 || len(se) > 0 {
			return false, errors.Annotatef(err, "systemctl check service %s failed", unitName)
		}
		return false, nil // Not found service with name unitName but not considered as error
	}
	soSplit := strings.Fields(string(so))
	if len(soSplit) < 1 {
		return false, errors.Annotatef(err, "systemctl check service %s wrong output", unitName)
	}

	return true, nil
}

// NewMockedVolume for unit test.
func NewMockedVolume() (Bot, *gfsmocks.Guestfs) {
	gfs := &gfsmocks.Guestfs{}

	vol := &bot{
		vol: &Volume{
			Volume: models.NewSysVolume(utils.GB, "unitest-image"),
		},
		newGuestfs: func(string) (guestfs.Guestfs, error) { return gfs, nil },
	}

	return vol, gfs
}
