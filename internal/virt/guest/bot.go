package guest

import (
	"context"
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/domain"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/virt/volume"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Bot .
type Bot interface { //nolint
	Close() error
	Create() error
	Boot() error
	Shutdown(force bool) error
	Suspend() error
	Resume() error
	Undefine() error
	Migrate() error
	OpenConsole(context.Context, types.OpenConsoleFlags) (types.Console, error)
	ExecuteCommand(context.Context, []string) (output []byte, exitCode, pid int, err error)
	GetState() (libvirt.DomainState, error)
	GetUUID() (string, error)
	IsFolder(context.Context, string) (bool, error)
	RemoveAll(context.Context, string) error
	Resize(cpu int, mem int64) error
	Capture(user, name string) (*models.UserImage, error)
	AmplifyVolume(vol volume.Virt, cap int64, devPath string) error
	AttachVolume(volmod *models.Volume, devName string) (rollback func(), err error)
	BindExtraNetwork() error
	OpenFile(path, mode string) (agent.File, error)
	MakeDirectory(ctx context.Context, path string, parent bool) error
	Trylock() error
	Unlock()
	CreateSnapshot(*models.Volume) error
	CommitSnapshot(*models.Volume, string) error
	CommitSnapshotByDay(*models.Volume, int) error
	RestoreSnapshot(*models.Volume, string) error
	CheckVolume(*models.Volume) error
	RepairVolume(*models.Volume) error
}

type bot struct {
	guest     *Guest
	virt      libvirt.Libvirt
	dom       domain.Domain
	ga        *agent.Agent
	flock     *utils.Flock
	newVolume func(*models.Volume) volume.Virt
}

func newVirtGuest(guest *Guest) (Bot, error) {
	virt, err := connectSystemLibvirt()
	if err != nil {
		return nil, errors.Trace(err)
	}

	vg := &bot{
		guest:     guest,
		virt:      virt,
		newVolume: newVolume,
	}
	vg.dom = domain.New(vg.guest.Guest, vg.virt)
	vg.flock = vg.newFlock()
	vg.ga = agent.New(vg.guest.SocketFilepath())

	return vg, nil
}

func connectSystemLibvirt() (libvirt.Libvirt, error) {
	return libvirt.Connect("qemu:///system")
}

func newVolume(volmod *models.Volume) volume.Virt {
	return volume.New(volmod)
}

func (v *bot) Close() (err error) {
	if _, err = v.virt.Close(); err != nil {
		log.WarnStack(err)
	}

	if err = v.ga.Close(); err != nil {
		log.WarnStack(err)
	}

	return
}

func (v *bot) Migrate() error {
	// TODO
	return nil
}

func (v *bot) Boot() error {
	return utils.Invoke([]func() error{
		v.dom.Boot,
		v.waitGA,
		v.setupNics,
		v.setupVols,
		v.execBatches,
		v.BindExtraNetwork,
	})
}

func (v *bot) waitGA() error {
	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GABootTimeout.Duration())
	defer cancel()

	for i := 1; ; i++ {
		if err := v.ga.Ping(ctx); err != nil {
			select {
			case <-ctx.Done():
				return errors.Trace(err)

			default:
				log.WarnStack(err)

				i %= 10
				time.Sleep(time.Second * time.Duration(i))

				if xe := v.reloadGA(); xe != nil {
					return errors.Wrap(err, xe)
				}

				continue
			}
		}

		return nil
	}
}

func (v *bot) Shutdown(force bool) error {
	return v.dom.Shutdown(force)
}

func (v *bot) Suspend() error {
	return v.dom.Suspend()
}

func (v *bot) Resume() error {
	return v.dom.Resume()
}

func (v *bot) Undefine() error {
	var undeVols = func() (err error) {
		v.guest.rangeVolumes(func(_ int, vol volume.Virt) bool {
			err = vol.Undefine()
			return err == nil
		})
		return
	}

	return utils.Invoke([]func() error{
		v.dom.Undefine,
		undeVols,
	})
}

func (v *bot) Create() error {
	return utils.Invoke([]func() error{
		v.allocVols,
		v.allocGuest,
	})
}

func (v *bot) allocVols() (err error) {
	v.guest.rangeVolumes(func(_ int, vol volume.Virt) bool {
		err = vol.Alloc(v.guest.Img)
		return err == nil
	})
	return
}

func (v *bot) allocGuest() error {
	if err := v.dom.Define(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (v *bot) CheckVolume(volmod *models.Volume) error {
	vol := v.newVolume(volmod)
	return vol.Check()
}

func (v *bot) RepairVolume(volmod *models.Volume) error {
	vol := v.newVolume(volmod)
	return vol.Repair()
}

func (v *bot) CreateSnapshot(volmod *models.Volume) error {
	vol := v.newVolume(volmod)
	return vol.CreateSnapshot()
}

func (v *bot) CommitSnapshot(volmod *models.Volume, snapID string) error {
	vol := v.newVolume(volmod)
	return vol.CommitSnapshot(snapID)
}

func (v *bot) CommitSnapshotByDay(volmod *models.Volume, day int) error {
	vol := v.newVolume(volmod)
	return vol.CommitSnapshotByDay(day)
}

func (v *bot) RestoreSnapshot(volmod *models.Volume, snapID string) error {
	vol := v.newVolume(volmod)
	return vol.RestoreSnapshot(snapID)
}

// AttachVolume .
func (v *bot) AttachVolume(volmod *models.Volume, devName string) (func(), error) {
	vol := v.newVolume(volmod)
	return vol.Attach(v.dom, v.ga, devName)
}

// AmplifyVolume .
func (v *bot) AmplifyVolume(vol volume.Virt, cap int64, devPath string) (err error) {
	_, err = vol.Amplify(cap, v.dom, v.ga, devPath)

	return err
}

func (v *bot) newFlock() *utils.Flock {
	var fn = fmt.Sprintf("%s.flock", v.guest.ID)
	var fpth = filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

func (v *bot) execBatches() error {
	for _, bat := range configs.Conf.Batches {
		if err := v.ga.ExecBatch(bat); err != nil {
			if bat.ForceOK {
				log.ErrorStackf(err, "forced batch error")
				metrics.IncrError()
				break
			}

			log.ErrorStackf(err, "non-forced batch err")
		}
	}

	// always non-error
	return nil
}

func (v *bot) setupVols() (err error) {
	v.guest.rangeVolumes(func(sn int, vol volume.Virt) bool {
		if vol.IsSys() {
			return true
		}
		err = vol.Mount(v.ga, vol.Model().GetDevicePathBySerialNumber(sn))
		return err == nil
	})
	return
}

func (v *bot) setupNics() error {
	var leng = time.Duration(len(v.guest.IPs))
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute*leng) //nolint
	defer cancel()

	if err := nic.NewNicList(v.guest.IPs, v.ga).Setup(ctx); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// BindExtraNetwork .
func (v *bot) BindExtraNetwork() error {
	dev := "eth0"
	distro := v.guest.Distro()

	if distro != types.Ubuntu {
		return nil
	}

	leng := time.Duration(v.guest.ExtraNetworks.Len())
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*leng) //nolint
	defer cancel()

	for _, netw := range v.guest.ExtraNetworks {
		// fn := fmt.Sprintf("%s.extra%d", dev, i)
		if err := nic.NewNic(netw.IP, v.ga).AddIP(ctx, dev); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (v *bot) reloadGA() error {
	if err := v.ga.Close(); err != nil {
		return errors.Trace(err)
	}

	v.ga = agent.New(v.guest.SocketFilepath())

	return nil
}

func (v *bot) OpenConsole(_ context.Context, flags types.OpenConsoleFlags) (types.Console, error) {
	ttyname, err := v.dom.GetConsoleTtyname()
	if err != nil {
		return nil, err
	}
	stream, err := v.openConsole(ttyname, flags)
	return stream, err
}

func (v *bot) openConsole(devname string, _ types.OpenConsoleFlags) (types.Console, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFL, syscall.FD_CLOEXEC)
	if errno != 0 {
		return nil, errors.Trace(err)
	}

	return fdAdapter{fd}, syscall.Connect(fd, &syscall.SockaddrUnix{Name: devname})
}

func (v *bot) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	var prog string
	var args []string

	switch leng := len(commands); {
	case leng < 1:
		return nil, -1, -1, errors.Annotatef(errors.ErrInvalidValue, "invalid command")
	case leng > 1:
		args = commands[1:]
		fallthrough
	default:
		prog = commands[0]
	}

	select {
	case <-ctx.Done():
		err = context.Canceled
		return

	case st := <-v.ga.ExecOutput(ctx, prog, args...):
		so, se, err := st.Stdio()
		return append(so, se...), st.Code, st.Pid, err
	}
}

func (v *bot) GetUUID() (string, error) {
	return v.dom.GetUUID()
}

func (v *bot) Capture(user, name string) (*models.UserImage, error) {
	if err := v.dom.CheckShutoff(); err != nil {
		return nil, errors.Trace(err)
	}

	vol, err := v.guest.sysVolume()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return vol.ConvertImage(user, name)
}

func (v *bot) Trylock() error {
	return v.flock.Trylock()
}

func (v *bot) Unlock() {
	v.flock.Close()
}

func (v *bot) GetState() (libvirt.DomainState, error) {
	return v.dom.GetState()
}

func (v *bot) Resize(cpu int, mem int64) error {
	return v.dom.SetSpec(cpu, mem)
}

// OpenFile .
func (v *bot) OpenFile(path string, mode string) (agent.File, error) {
	return agent.OpenFile(v.ga, path, mode)
}

// MakeDirectory .
func (v *bot) MakeDirectory(ctx context.Context, path string, parent bool) error {
	return v.ga.MakeDirectory(ctx, path, parent)
}

// IsFolder .
func (v *bot) IsFolder(ctx context.Context, path string) (bool, error) {
	return v.ga.IsFolder(ctx, path)
}

// RemoveAll .
func (v *bot) RemoveAll(ctx context.Context, path string) error {
	return v.ga.RemoveAll(ctx, path)
}
