package guest

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/domain"
	"github.com/projecteru2/yavirt/internal/virt/nic"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	vmiFact "github.com/yuyang0/vmimage/factory"
	vmitypes "github.com/yuyang0/vmimage/types"
)

// Bot .
type Bot interface { //nolint
	Trylock() error
	Unlock()
	Close() error
	Define(ctx context.Context) error
	Undefine() error

	Boot(ctx context.Context) error
	Shutdown(ctx context.Context, force bool) error
	Suspend() error
	Resume() error
	Resize(cpu int, mem int64) error

	Migrate() error
	OpenConsole(context.Context, types.OpenConsoleFlags) (*libvirt.Console, error)
	ExecuteCommand(context.Context, []string) (output []byte, exitCode, pid int, err error)
	GetState() (libvirt.DomainState, error)
	GetUUID() (string, error)
	Capture(imgName string) (*vmitypes.Image, error)
	BindExtraNetwork() error

	// fs-related functions
	IsFolder(context.Context, string) (bool, error)
	RemoveAll(context.Context, string) error
	OpenFile(ctx context.Context, path, mode string) (agent.File, error)
	MakeDirectory(ctx context.Context, path string, parent bool) error
	FSFreezeAll(ctx context.Context) (int, error)
	FSThawAll(ctx context.Context) (int, error)
	FSFreezeStatus(ctx context.Context) (string, error)

	// GPU-related functions
	AttachGPUs(pcm map[string]int) error
	DetachGPUs(pcm map[string]int) error

	// storage-related functions
	ReplaceSysVolume(vol volume.Volume) error
	AmplifyVolume(vol volume.Volume, delta int64) error
	AttachVolume(volmod volume.Volume) (rollback func(), err error)
	DetachVolume(vol volume.Volume) (err error)
	CheckVolume(volume.Volume) error
	RepairVolume(volume.Volume) error
	CreateSnapshot(volume.Volume) error
	CommitSnapshot(volume.Volume, string) error
	CommitSnapshotByDay(volume.Volume, int) error
	RestoreSnapshot(volume.Volume, string) error
}

type bot struct {
	guest *Guest
	virt  libvirt.Libvirt
	dom   domain.Domain
	ga    *agent.Agent
	flock *utils.Flock
}

func newVirtGuest(guest *Guest) (Bot, error) {
	virt, err := connectSystemLibvirt()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	vg := &bot{
		guest: guest,
		virt:  virt,
	}
	vg.dom = domain.New(vg.guest.Guest, vg.virt)
	vg.flock = vg.newFlock()

	vg.ga = agent.New(guest.ID, virt)

	return vg, nil
}

func connectSystemLibvirt() (libvirt.Libvirt, error) {
	return libvirt.Connect("qemu:///system")
}

func (v *bot) Close() (err error) {
	logger := log.WithFunc("Close").WithField("guest", v.guest.ID)
	if _, err = v.virt.Close(); err != nil {
		logger.Warnf(context.TODO(), "virt close error:%s", err)
	}

	if err = v.ga.Close(); err != nil {
		logger.Warnf(context.TODO(), "ga close error:%s", err)
	}

	return
}

func (v *bot) Migrate() error {
	// TODO
	return nil
}

func (v *bot) Boot(ctx context.Context) error {
	logger := log.WithFunc("Boot").WithField("guest", v.guest.ID)

	logger.Infof(ctx, "Boot: stage1 -> Domain boot...")
	if err := v.dom.Boot(ctx); err != nil {
		return err
	}

	logger.Infof(ctx, "Boot: stage2 -> Waiting GA...")
	if err := v.waitGA(ctx); err != nil {
		return err
	}

	// the following operations only log error and don't return error

	// In normal case, we use cloud-init to set NICs, so here is just a fallback
	logger.Info(ctx, "Boot: stage3 -> Setting NICs...")
	if err := v.setupNics(ctx); err != nil {
		logger.Error(ctx, err, "Boot: stage3 -> Setting NICs failed")
	}

	logger.Info(ctx, "Boot: stage4 -> Setting Vols...")
	if err := v.setupVols(); err != nil {
		logger.Error(ctx, err, "Boot: stage4 -> Setting Vols failed")
	}
	logger.Info(ctx, "Boot: stage5 -> Executing Batches...")
	if err := v.execBatches(); err != nil {
		logger.Errorf(ctx, err, "Boot: stage5 -> Executing Batches failed")
	}
	logger.Info(ctx, "Boot: stage6 -> Binding extra networks...")
	if err := v.BindExtraNetwork(); err != nil {
		logger.Error(ctx, err, "Boot: stage6 -> Binding extra networks failed")
	}
	return nil
}

func (v *bot) waitGA(ctx context.Context) error {
	logger := log.WithFunc("waitGA").WithField("guest", v.guest.ID)
	// Create a new context with a shorter timeout
	// so that we can return a more informative error message when timeout.
	timeout := 7 * time.Minute
	if daedline, ok := ctx.Deadline(); ok {
		timeout = time.Until(daedline) - 30*time.Second
		if timeout < 0 {
			timeout = time.Until(daedline)
		}
	}
	newCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for i := 0; ; i++ {
		err := v.ga.Ping(newCtx)
		if err == nil {
			return nil
		}

		logger.Warnf(newCtx, "[waitGA] ping %d times, but still failed %s", i+1, err.Error())

		timeout := time.Duration((i%10)+1) * time.Second

		select {
		case <-newCtx.Done():
			errmsg := `
timeout when waiting boot, this can be caused by:
1. image doesn't contain qemu-guest-agent package.
2. image or system disk is corrupted and can't be booted normally.
			`
			return errors.Wrap(err, errmsg)
		case <-time.After(timeout):
			if xe := v.reloadGA(); xe != nil {
				return errors.CombineErrors(err, xe)
			}
		}
	}
}

func (v *bot) Shutdown(ctx context.Context, force bool) error {
	return v.dom.Shutdown(ctx, force)
}

func (v *bot) Suspend() error {
	return v.dom.Suspend()
}

func (v *bot) Resume() error {
	return v.dom.Resume()
}

func (v *bot) Undefine() error {
	return v.dom.Undefine()
}

func (v *bot) Define(_ context.Context) (err error) {
	if err := v.dom.Define(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (v *bot) CheckVolume(volmod volume.Volume) error {
	return volFact.Check(volmod)
}

func (v *bot) RepairVolume(volmod volume.Volume) error {
	return volFact.Repair(volmod)
}

func (v *bot) CreateSnapshot(volmod volume.Volume) error {
	return volFact.CreateSnapshot(volmod)
}

func (v *bot) CommitSnapshot(volmod volume.Volume, snapID string) error {
	return volFact.CommitSnapshot(volmod, snapID)
}

func (v *bot) CommitSnapshotByDay(volmod volume.Volume, day int) error {
	return volFact.CommitSnapshotByDay(volmod, day)
}

func (v *bot) RestoreSnapshot(volmod volume.Volume, snapID string) error {
	return volFact.RestoreSnapshot(volmod, snapID)
}

// AttachVolume .
func (v *bot) AttachVolume(vol volume.Volume) (rollback func(), err error) {
	devName := vol.GetDevice()
	dom, err := v.dom.Lookup()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	rollback, err = volFact.Create(vol)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	defer func() {
		if err != nil {
			rollback()
		}
		rollback = nil
	}()

	var st libvirt.DomainState
	buf, err := vol.GenerateXML()
	if err != nil {
		return
	}
	st, err = dom.AttachDevice(string(buf))
	if err == nil && st == libvirt.DomainRunning && configs.Conf.Storage.InitGuestVolume {
		log.Debugf(context.TODO(), "Mount(%s): start to mount volume(%s)", v.guest.ID, vol.GetMountDir())
		err = volFact.Mount(vol, v.ga, base.GetDevicePathByName(devName))
	}
	return
}

// DetachVolume .
func (v *bot) DetachVolume(vol volume.Volume) (err error) {
	logger := log.WithFunc("DetachVolume")
	devPath := vol.GetDevice()
	if configs.Conf.Storage.InitGuestVolume {
		switch st, err := v.GetState(); {
		case err != nil:
			return errors.Wrap(err, "")
		case st == libvirt.DomainRunning:
			if err := volFact.Unmount(vol, v.ga, devPath); err != nil {
				return errors.Wrap(err, "")
			}
		default:
			logger.Warnf(context.TODO(), "the guest is not running, so ignore to umount")
		}
	}
	_, err = v.dom.DetachVolume(devPath)
	return
}

func (v *bot) ReplaceSysVolume(vol volume.Volume) error {
	diskXML, err := vol.GenerateXML()
	if err != nil {
		return err
	}
	return v.dom.ReplaceSysVolume(string(diskXML))
}

// AmplifyVolume .
func (v *bot) AmplifyVolume(vol volume.Volume, delta int64) (err error) {
	devPath := base.GetDevicePathByName(vol.GetDevice())
	dom, err := v.dom.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}
	_, err = volFact.Amplify(vol, delta, dom, v.ga, devPath)

	return err
}

func (v *bot) AttachGPUs(pcm map[string]int) error {
	for prod, count := range pcm {
		if _, err := v.dom.AttachGPU(prod, count); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (v *bot) DetachGPUs(pcm map[string]int) error {
	for prod, count := range pcm {
		if _, err := v.dom.DetachGPU(prod, count); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (v *bot) newFlock() *utils.Flock {
	var fn = fmt.Sprintf("guest_%s.flock", v.guest.ID)
	var fpth = filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

func (v *bot) execBatches() error { //nolint:unparam
	logger := log.WithFunc("execBatches").WithField("guest", v.guest.ID)
	for _, bat := range configs.Conf.Batches {
		if err := v.ga.ExecBatch(bat); err != nil {
			if bat.ForceOK {
				logger.Errorf(context.TODO(), err, "forced batch error")
				metrics.IncrError()
				break
			}

			logger.Errorf(context.TODO(), err, "non-forced batch err")
		}
	}

	// always non-error
	return nil
}

func (v *bot) setupVols() (err error) {
	if !configs.Conf.Storage.InitGuestVolume {
		return nil
	}
	v.guest.rangeVolumes(func(sn int, vol volume.Volume) bool {
		if vol.IsSys() {
			return true
		}
		err = volFact.Mount(vol, v.ga, base.GetDevicePathBySerialNumber(sn))
		return err == nil
	})
	return
}

func (v *bot) setupNics(ctx context.Context) error {
	if err := nic.NewNicList(v.guest.IPs, v.ga).Setup(ctx); err != nil {
		return errors.Wrap(err, "")
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
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func (v *bot) reloadGA() error {
	if err := v.ga.Close(); err != nil {
		return errors.Wrap(err, "")
	}

	v.ga = agent.New(v.guest.ID, v.virt)

	return nil
}

func (v *bot) OpenConsole(_ context.Context, flags types.OpenConsoleFlags) (*libvirt.Console, error) {
	err := v.dom.CheckRunning()
	if err != nil {
		return nil, err
	}
	// yavirtctl may specify devname directly
	ttyname := flags.Devname
	if ttyname == "" {
		ttyname, err = v.dom.GetConsoleTtyname()
		if err != nil {
			return nil, err
		}
	}
	c, err := v.dom.OpenConsole(ttyname, flags)
	return c, err
}

func (v *bot) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	var prog string
	var args []string

	switch leng := len(commands); {
	case leng < 1:
		return nil, -1, -1, errors.Wrapf(terrors.ErrInvalidValue, "invalid command")
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

func (v *bot) Capture(imgName string) (uimg *vmitypes.Image, err error) {
	if err := v.dom.CheckShutoff(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	vol, err := v.guest.sysVolume()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if err := vol.Lock(); err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer vol.Unlock()

	if !vol.IsSys() {
		return nil, errors.Wrapf(terrors.ErrNotSysVolume, "%s is not a system volume", vol.GetID())
	}

	if uimg, err = vol.CaptureImage(imgName); err != nil {
		return nil, errors.Wrapf(err, "failed to capture image %s", imgName)
	}
	defer func() {
		if err != nil {
			_ = vmiFact.RemoveLocal(context.TODO(), uimg)
		}
	}()

	return uimg, nil
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
func (v *bot) OpenFile(ctx context.Context, path string, mode string) (agent.File, error) {
	return agent.OpenFile(ctx, v.ga, path, mode)
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

func (v *bot) FSFreezeAll(ctx context.Context) (int, error) {
	return v.ga.FSFreezeAll(ctx)
}

func (v *bot) FSThawAll(ctx context.Context) (int, error) {
	return v.ga.FSThawAll(ctx)
}

func (v *bot) FSFreezeStatus(ctx context.Context) (string, error) {
	return v.ga.FSFreezeStatus(ctx)
}
