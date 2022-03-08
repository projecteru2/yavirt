package guest

import (
	"context"
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/metric"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt/agent"
	"github.com/projecteru2/yavirt/virt/domain"
	"github.com/projecteru2/yavirt/virt/nic"
	"github.com/projecteru2/yavirt/virt/types"
	"github.com/projecteru2/yavirt/virt/volume"
)

// Bot .
type Bot interface {
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
	Capture(user, name string) (*model.UserImage, error)
	AmplifyVolume(vol volume.Virt, cap int64, devPath string) error
	AttachVolume(volmod *model.Volume, devName string) (rollback func(), err error)
	BindExtraNetwork() error
	OpenFile(path, mode string) (agent.File, error)
	MakeDirectory(ctx context.Context, path string, parent bool) error
	Trylock() error
	Unlock()
	CreateSnapshot(*model.Volume) error
	CommitSnapshot(*model.Volume, string) error
	CommitSnapshotByDay(*model.Volume, int) error
	RestoreSnapshot(*model.Volume, string) error
	CheckVolume(*model.Volume) error
	RepairVolume(*model.Volume) error
}

type bot struct {
	guest     *Guest
	virt      libvirt.Libvirt
	dom       domain.Domain
	ga        *agent.Agent
	flock     *util.Flock
	newVolume func(*model.Volume) volume.Virt
}

func newVirtGuest(guest *Guest) (Bot, error) {
	virt, err := libvirt.Connect("qemu:///system")
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

func newVolume(volmod *model.Volume) volume.Virt {
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
	return util.Invoke([]func() error{
		v.dom.Boot,
		v.waitGA,
		v.setupNics,
		v.setupVols,
		v.execBatches,
		v.BindExtraNetwork,
	})
}

func (v *bot) waitGA() error {
	var ctx, cancel = context.WithTimeout(context.Background(), config.Conf.GABootTimeout.Duration())
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

	return util.Invoke([]func() error{
		v.dom.Undefine,
		undeVols,
	})
}

func (v *bot) Create() error {
	return util.Invoke([]func() error{
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

func (v *bot) CheckVolume(volmod *model.Volume) error {
	vol := v.newVolume(volmod)
	return vol.Check()
}

func (v *bot) RepairVolume(volmod *model.Volume) error {
	vol := v.newVolume(volmod)
	return vol.Repair()
}

func (v *bot) CreateSnapshot(volmod *model.Volume) error {
	vol := v.newVolume(volmod)
	return vol.CreateSnapshot()
}

func (v *bot) CommitSnapshot(volmod *model.Volume, snapID string) error {
	vol := v.newVolume(volmod)
	return vol.CommitSnapshot(snapID)
}

func (v *bot) CommitSnapshotByDay(volmod *model.Volume, day int) error {
	vol := v.newVolume(volmod)
	return vol.CommitSnapshotByDay(day)
}

func (v *bot) RestoreSnapshot(volmod *model.Volume, snapID string) error {
	vol := v.newVolume(volmod)
	return vol.RestoreSnapshot(snapID)
}

// AttachVolume .
func (v *bot) AttachVolume(volmod *model.Volume, devName string) (func(), error) {
	vol := v.newVolume(volmod)
	return vol.Attach(v.dom, v.ga, devName)
}

// AmplifyVolume .
func (v *bot) AmplifyVolume(vol volume.Virt, cap int64, devPath string) (err error) {
	_, err = vol.Amplify(cap, v.dom, v.ga, devPath)

	return err
}

func (v *bot) newFlock() *util.Flock {
	var fn = fmt.Sprintf("%s.flock", v.guest.ID)
	var fpth = filepath.Join(config.Conf.VirtFlockDir, fn)
	return util.NewFlock(fpth)
}

func (v *bot) execBatches() error {
	for _, bat := range config.Conf.Batches {
		if err := v.ga.ExecBatch(bat); err != nil {
			if bat.ForceOK {
				log.ErrorStackf(err, "forced batch error")
				metric.IncrError()
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
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute*leng)
	defer cancel()

	for i, ip := range v.guest.IPs {
		var dev = fmt.Sprintf("eth%d", i)
		var distro = v.guest.Distro()

		if err := nic.NewNic(ip, v.ga).Setup(ctx, distro, dev); err != nil {
			return errors.Trace(err)
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*leng)
	defer cancel()

	for i, netw := range v.guest.ExtraNetworks {
		fn := fmt.Sprintf("%s.extra%d", dev, i)
		if err := nic.NewNic(netw.IP, v.ga).AddIP(ctx, distro, dev, fn); err != nil {
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

func (v *bot) OpenConsole(ctx context.Context, flags types.OpenConsoleFlags) (types.Console, error) {
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

func (v *bot) Capture(user, name string) (*model.UserImage, error) {
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
