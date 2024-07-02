package guest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/vmcache"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
	vmiFact "github.com/yuyang0/vmimage/factory"
	vmitypes "github.com/yuyang0/vmimage/types"
)

// Guest .
type Guest struct {
	*models.Guest

	newBot func(*Guest) (Bot, error)
}

// New initializes a new Guest.
func New(_ context.Context, g *models.Guest) *Guest {
	return &Guest{
		Guest:  g,
		newBot: newVirtGuest,
	}
}

// ListLocalIDs lists all local guest domain names.
func ListLocalIDs(ctx context.Context) ([]string, error) {
	virt, err := connectSystemLibvirt()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	defer func() {
		if _, ce := virt.Close(); ce != nil {
			log.WithFunc("ListLocalIDs").Errorf(ctx, ce, "failed to close virt")
		}
	}()

	return virt.ListDomainsNames()
}

// Load .
func (g *Guest) Load(opts ...models.Option) error {
	host, err := models.LoadHost()
	if err != nil {
		return errors.Wrap(err, "")
	}

	hand, err := g.NetworkHandler()
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := g.Guest.Load(host, hand, opts...); err != nil {
		return errors.Wrap(err, "")
	}

	return g.loadExtraNetworks()
}

// SyncState .
func (g *Guest) SyncState(ctx context.Context) error {
	switch g.Status {
	case meta.StatusDestroying:
		return g.ProcessDestroy(ctx, false)

	case meta.StatusStopping, meta.StatusStopped:
		return g.stop(ctx, true)

	case meta.StatusRunning, meta.StatusStarting:
		return g.start(ctx)

	case meta.StatusCreating:
		return g.create(ctx)

	default:
		// nothing to do
		return nil
	}
}

func (g *Guest) UpdateStateIfNecessary() error {
	err := g.botOperate(func(bot Bot) error { //nolint
		// check if there is inconsistent status
		dce := vmcache.FetchDomainEntry(g.ID)
		if dce != nil {
			switch {
			case (g.Status == meta.StatusRunning) && dce.IsStopped():
				_ = g.ForwardStatus(meta.StatusStopped, true)
			case (g.Status == meta.StatusStopped) && dce.IsRunning():
				_ = g.ForwardStatus(meta.StatusRunning, true)
			}
		}
		return nil
	})
	if err == nil || errors.Is(err, terrors.ErrFlockLocked) {
		return nil
	}
	return err
}

// Start .
func (g *Guest) Start(ctx context.Context, force bool) error {
	if err := g.ForwardStarting(force); err != nil {
		return err
	}
	defer log.Debugf(ctx, "exit g.Start")
	return g.start(ctx)
}

func (g *Guest) start(ctx context.Context) error {
	return g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Wrap(err, "")
		case st == libvirt.DomainRunning:
			return g.ForwardRunning()
		}
		log.Debugf(ctx, "Entering Boot")
		if err := bot.Boot(ctx); err != nil {
			return err
		}
		log.Debugf(ctx, "Entering joinEthernet")
		if err := g.joinEthernet(); err != nil {
			return err
		}
		log.Debugf(ctx, "Limiting bandwidth")
		if err := g.limitBandwidth(); err != nil {
			return err
		}
		log.Debugf(ctx, "Entering forwardRunning")
		return g.ForwardRunning()
	})
}

// Resize .
func (g *Guest) Resize(cpumem *cpumemtypes.EngineParams, gpu *gputypes.EngineParams, vols []volume.Volume) error {
	// Only checking, without touch metadata
	// due to we wanna keep further booting successfully all the time.
	if !g.CheckForwardStatus(meta.StatusResizing) {
		return errors.Wrapf(terrors.ErrForwardStatus, "only stopped/running guest can be resized, but it's %s", g.Status)
	}

	newVolMap := map[string]volume.Volume{}
	for _, vol := range vols {
		newVolMap[vol.GetMountDir()] = vol
	}
	if !cpumem.Remap {
		// Actually, mntCaps from ERU will include completed volumes,
		// even those volumes aren't affected.
		if err := g.handleResizeVolumes(newVolMap); err != nil {
			return errors.Wrap(err, "")
		}
		if err := g.handleResizeGPU(gpu); err != nil {
			return errors.Wrap(err, "")
		}
	}

	log.WithFunc("Guest.Resize").Infof(context.TODO(), "Resize(%s): Resize cpu and memory if necessary", g.ID)
	if int(cpumem.CPU) == g.CPU && cpumem.Memory == g.Memory {
		return nil
	}

	return g.resizeSpec(int(cpumem.CPU), cpumem.Memory)
}

func (g *Guest) handleResizeGPU(eParams *gputypes.EngineParams) (err error) {
	addDiff := eParams.DeepCopy()
	addDiff.Sub(g.GPUEngineParams)
	subDiff := g.GPUEngineParams.DeepCopy()
	subDiff.Sub(eParams)
	needUpdate := false
	if addDiff.Count() > 0 {
		// attach new GPU
		err = g.botOperate(func(bot Bot) error {
			return bot.AttachGPUs(addDiff.ProdCountMap)
		})
		if err != nil {
			return
		}
		needUpdate = true
	}
	if subDiff.Count() > 0 {
		// detach old GPU
		err = g.botOperate(func(bot Bot) error {
			return bot.DetachGPUs(subDiff.ProdCountMap)
		})
		if err != nil {
			return
		}
		needUpdate = true
	}
	if needUpdate {
		g.GPUEngineParams = eParams
		err = g.Save()
	}
	return
}

func (g *Guest) handleResizeVolumes(newVolMap map[string]volume.Volume) error {
	existVolMap := make(map[string]volume.Volume)
	for _, existVol := range g.Vols {
		existVolMap[existVol.GetMountDir()] = existVol
		if existVol.IsSys() {
			continue
		}
		if _, ok := newVolMap[existVol.GetMountDir()]; !ok {
			if err := g.detachVol(existVol); err != nil {
				return errors.Wrap(err, "")
			}
		}
	}
	for mountDir, newVol := range newVolMap {
		existVol, ok := existVolMap[mountDir]
		if !ok { //nolint
			if err := g.attachVol(newVol); err != nil {
				return errors.Wrap(err, "")
			}
		} else {
			if newVol.GetSize() > 0 {
				if err := g.amplifyOrigVol(existVol, newVol.GetSize()); err != nil {
					return errors.Wrap(err, "")
				}
			} else {
				if err := g.detachVol(existVol); err != nil {
					return errors.Wrap(err, "")
				}
			}
		}
	}
	return nil
}

func (g *Guest) amplifyOrigVol(existVol volume.Volume, expectSize int64) error {
	ctx := context.TODO()

	log.Infof(ctx, "[amplifyOrigVol] Amplifying volume %s(device:%s, guest %s)", existVol.GetID(), existVol.GetDevice(), g.ID)
	var err error
	delta := expectSize - existVol.GetSize()
	if delta <= 0 {
		if delta < 0 {
			log.Warnf(ctx, "[amplifyOrigVol] Don't allow to shrink volume(%s)", existVol.GetID())
		}
		return nil
	}

	err = g.botOperate(func(bot Bot) error {
		return bot.AmplifyVolume(existVol, delta)
	})
	if err != nil {
		return err
	}
	return nil
}

func (g *Guest) detachVol(existVol volume.Volume) error {
	ctx := context.TODO()
	logger := log.WithFunc("detachVol").WithField("guest", g.ID)

	var err error
	logger.Infof(ctx, "Detaching volume %v(device:%s, guest %s)", existVol, existVol.GetDevice(), g.ID)
	err = g.botOperate(func(bot Bot) error {
		return bot.DetachVolume(existVol)
	})
	if err != nil {
		logger.Errorf(ctx, err, "Failed to detach volume(%s)", existVol.GetID())
		return err
	}
	g.RemoveVol(existVol.GetID())
	err = existVol.Delete(true)
	if err != nil {
		logger.Errorf(ctx, err, "Failed to delete volume in etcd(%s)", existVol.GetID())
		return err
	}
	err = g.Save()
	if err != nil {
		logger.Errorf(ctx, err, "Failed to detach volume(%s)", existVol.GetID())
		return err
	}
	return err
}

func (g *Guest) attachVol(vol volume.Volume) (err error) {
	devName := g.nextVolumeName()

	vol.SetGuestID(g.ID)
	vol.SetStatus(g.Status, true) //nolint:errcheck
	vol.GenerateID()
	vol.SetDevice(devName)
	log.Infof(context.TODO(), "[attachVol] Attaching volume %v(device: %s, guest %s)", vol, vol.GetDevice(), g.ID)
	if err = g.AppendVols(vol); err != nil {
		return errors.Wrap(err, "")
	}

	var rollback func()
	defer func() {
		if err != nil {
			if rollback != nil {
				rollback()
			}
			g.RemoveVol(vol.GetID())
		}
	}()

	if err = g.botOperate(func(bot Bot) (ae error) {
		rollback, ae = bot.AttachVolume(vol)
		return ae
	}); err != nil {
		return
	}

	return g.Save()
}

func (g *Guest) resizeSpec(cpu int, mem int64) error {
	if err := g.botOperate(func(bot Bot) error {
		return bot.Resize(cpu, mem)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return g.Guest.Resize(cpu, mem)
}

// ListSnapshot If volID == "", list snapshots of all vols. Else will find vol with matching volID.
func (g *Guest) ListSnapshot(volID string) (map[volume.Volume]base.Snapshots, error) {
	volSnap := make(map[volume.Volume]base.Snapshots)

	matched := false
	for _, v := range g.Vols {
		if v.GetID() == volID || volID == "" {
			api := v.NewSnapshotAPI()
			volSnap[v] = api.List()
			matched = true
		}
	}

	if !matched {
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "volID %s not exists", volID)
	}

	return volSnap, nil
}

// CheckVolume .
func (g *Guest) CheckVolume(volID string) error {
	if g.Status != meta.StatusStopped && g.Status != meta.StatusPaused {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only paused/stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CheckVolume(vol)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// RepairVolume .
func (g *Guest) RepairVolume(volID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RepairVolume(vol)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// CreateSnapshot .
func (g *Guest) CreateSnapshot(volID string) error {
	if g.Status != meta.StatusStopped && g.Status != meta.StatusPaused {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only paused/stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CreateSnapshot(vol)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshot(volID string, snapID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshot(vol, snapID)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshotByDay(volID string, day int) error {
	if g.Status != meta.StatusStopped {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshotByDay(vol, day)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// RestoreSnapshot .
func (g *Guest) RestoreSnapshot(volID string, snapID string) error {
	if g.Status != meta.StatusStopped {
		return errors.Wrapf(terrors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RestoreSnapshot(vol, snapID)
	}); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}

// Capture .
func (g *Guest) Capture(imgName string, overridden bool) (uimg *vmitypes.Image, err error) {
	if err = g.ForwardCapturing(); err != nil {
		return
	}

	if err = g.botOperate(func(bot Bot) error {
		var ce error
		if uimg, ce = bot.Capture(imgName); ce != nil {
			return errors.Wrap(ce, "Failed to capture image")
		}
		return g.ForwardCaptured()
	}); err != nil {
		return
	}

	if err = g.ForwardStopped(false); err != nil {
		return
	}
	rc, err := vmiFact.Push(context.TODO(), uimg, overridden)
	if err != nil {
		return
	}
	defer interutils.EnsureReaderClosed(rc)

	return uimg, err
}

// Migrate .
func (g *Guest) Migrate() error {
	return utils.Invoke([]func() error{
		g.ForwardMigrating,
		g.migrate,
	})
}

func (g *Guest) migrate() error {
	return g.botOperate(func(bot Bot) error {
		return bot.Migrate()
	})
}

func (g *Guest) PrepareVolumesForCreate(ctx context.Context) error {
	rl := interutils.GetRollbackListFromContext(ctx)
	for _, vol := range g.Vols {
		if err := volume.WithLocker(vol, func() error {
			if vol.IsSys() {
				return vol.PrepareSysDisk(ctx, g.Img)
			}
			return vol.PrepareDataDisk(ctx)
		}); err != nil {
			return err
		}
		if rl != nil {
			rl.Append(func() error { return volFact.Undefine(vol) }, "dealloc volume")
		}
	}
	return nil
}

// DefineGuestForCreate .
func (g *Guest) DefineGuestForCreate(ctx context.Context) error {
	if err := g.ForwardCreating(); err != nil {
		return err
	}
	rl := interutils.GetRollbackListFromContext(ctx)
	// add a rollback function here, so
	if rl != nil {
		rl.Append(func() error {
			return g.botOperate(func(bot Bot) error {
				return bot.Undefine()
			}, true)
		}, "Undefine guest")
	}
	return g.create(ctx)
}

func (g *Guest) create(ctx context.Context) error {
	return g.botOperate(func(bot Bot) error {
		return bot.Define(ctx)
	})
}

// Stop .
func (g *Guest) Stop(ctx context.Context, force bool) error {
	if err := g.ForwardStopping(); !force && err != nil {
		return errors.Wrap(err, "")
	}
	return g.stop(ctx, force)
}

func (g *Guest) stop(ctx context.Context, force bool) error {
	return g.botOperate(func(bot Bot) error {
		if err := bot.Shutdown(ctx, force); err != nil {
			return errors.Wrap(err, "")
		}
		return g.ForwardStopped(force)
	})
}

// Suspend .
func (g *Guest) Suspend() error {
	return utils.Invoke([]func() error{
		g.ForwardPausing,
		g.suspend,
	})
}

func (g *Guest) suspend() error {
	return g.botOperate(func(bot Bot) error {
		return utils.Invoke([]func() error{
			bot.Suspend,
			g.ForwardPaused,
		})
	})
}

// Resume .
func (g *Guest) Resume() error {
	return utils.Invoke([]func() error{
		g.ForwardResuming,
		g.resume,
	})
}

func (g *Guest) resume() error {
	return g.botOperate(func(bot Bot) error {
		return utils.Invoke([]func() error{
			bot.Resume,
			g.ForwardRunning,
		})
	})
}

// rewrite the sys disk with image
func (g *Guest) InitSysDisk(
	ctx context.Context, img *vmitypes.Image,
	args *types.InitSysDiskArgs, newSysVol volume.Volume,
) error {
	logger := log.WithFunc("InitSysDisk")
	ciCfg, err := g.GenCloudInit()
	if err != nil {
		return errors.Wrap(err, "failed to generate cloud init config")
	}
	var (
		ciUpdated bool
	)
	if args.Username != "" {
		ciCfg.Username = args.Username
		ciUpdated = true
	}
	if args.Password != "" {
		ciCfg.Password = args.Password
		ciUpdated = true
	}
	if ciUpdated {
		bs, _ := json.Marshal(ciCfg)
		g.JSONLabels["instance/cloud-init"] = string(bs)
	}
	if g.ImageName != img.Fullname() {
		g.ImageName = img.Fullname()
		g.Img = img
	}
	return g.botOperate(func(bot Bot) error {
		if err := bot.Shutdown(ctx, true); err != nil {
			return errors.Wrap(err, "")
		}
		if err := g.ForwardStopped(true); err != nil {
			return errors.Wrap(err, "")
		}
		oldSysVol := g.Vols[0]
		newSysVol.SetDevice(oldSysVol.GetDevice())
		newSysVol.SetGuestID(g.ID)
		newSysVol.SetHostname(g.HostName)
		newSysVol.GenerateID()
		newSysVol.SetStatus(g.Status, true) //nolint:errcheck
		if err := newSysVol.Save(); err != nil {
			logger.Errorf(ctx, err, "failed to save new system volume: %v", newSysVol)
			return errors.Wrapf(err, "failed to save new system volume")
		}
		logger.Infof(ctx, "new system volume: %v", newSysVol)
		// create new sys disk and write image to it
		if err := newSysVol.PrepareSysDisk(ctx, img); err != nil {
			return errors.Wrap(err, "")
		}

		if err := g.SwitchVol(newSysVol, 0); err != nil {
			return errors.Wrap(err, "")
		}

		// remove local or rbd disk
		if err := oldSysVol.Cleanup(); err != nil {
			return errors.Wrap(err, "")
		}
		if err := oldSysVol.Delete(true); err != nil {
			return errors.Wrap(err, "")
		}

		if ciUpdated {
			output := filepath.Join(configs.Conf.VirtCloudInitDir, fmt.Sprintf("%s.iso", g.ID))
			if err := ciCfg.ReplaceUserData(output); err != nil {
				return errors.Wrap(err, "")
			}
		}
		// change domain xml
		if err := bot.ReplaceSysVolume(newSysVol); err != nil {
			return errors.Wrapf(err, "failed to undefine domain when init sys disk")
		}
		if err := g.Save(); err != nil {
			return errors.Wrap(err, "")
		}
		return nil
	})
}

// Destroy .
func (g *Guest) Destroy(ctx context.Context, force bool) (<-chan error, error) {
	if err := g.Stop(ctx, force); err != nil && !terrors.IsDomainNotExistsErr(err) {
		return nil, errors.Wrap(err, "")
	}
	if err := g.ForwardDestroying(force); err != nil {
		return nil, errors.Wrap(err, "")
	}
	log.Infof(ctx, "[guest.Destroy] set state of guest %s to destroying", g.ID)

	done := make(chan error, 1)

	// will return immediately as the destroy request has been accepted.
	// the detail work will be processed asynchronously
	go func() {
		err := g.ProcessDestroy(ctx, force)
		if err != nil {
			log.Errorf(ctx, err, "destroy guest %s failed", g.ID)
			//TODO: move to recovery list
		}
		done <- err
	}()

	return done, nil
}

func (g *Guest) ProcessDestroy(ctx context.Context, force bool) error {
	logger := log.WithFunc("Guest.ProcessDestroy").WithField("guest", g.ID)
	logger.Infof(ctx, "begin to destroy guest")
	return g.botOperate(func(bot Bot) error {
		if err := bot.Undefine(); err != nil {
			logger.Errorf(ctx, err, "failed to undefine guest")
			return errors.Wrap(err, "")
		}
		// delete cloud-init iso
		ciISOFname := filepath.Join(configs.Conf.VirtCloudInitDir, fmt.Sprintf("%s.iso", g.ID))
		_ = os.Remove(ciISOFname)

		// try best behavior
		if err := g.DeleteNetwork(); err != nil {
			logger.Errorf(ctx, err, "failed to delete network")
		}

		for _, vol := range g.Vols {
			// try best behavior
			if err := volFact.Undefine(vol); err != nil {
				logger.Errorf(ctx, err, "failed to undefine volume (volID: %s)", vol.GetID())
			}
		}
		return g.Delete(force)
	}, force)
}

func (g *Guest) FSFreezeAll(ctx context.Context) (nFS int, err error) {
	err = g.botOperate(func(bot Bot) error {
		var err error
		nFS, err = bot.FSFreezeAll(ctx)
		return err
	})
	return
}

func (g *Guest) FSThawAll(ctx context.Context) (nFS int, err error) {
	err = g.botOperate(func(bot Bot) error {
		var err error
		nFS, err = bot.FSThawAll(ctx)
		return err
	})
	return
}

func (g *Guest) FSFreezeStatus(ctx context.Context) (status string, err error) {
	err = g.botOperate(func(bot Bot) error {
		var err error
		status, err = bot.FSFreezeStatus(ctx)
		return err
	})
	return
}

const waitRetries = 30 // 30 second
// Wait .
func (g *Guest) Wait(toStatus string, block bool) error {
	if !g.CheckForwardStatus(toStatus) {
		return terrors.ErrForwardStatus
	}
	return g.botOperate(func(bot Bot) error { //nolint:revive
		cnt := 0
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if g.Status == toStatus {
				return nil
			}
			if !block {
				if cnt++; cnt > waitRetries {
					return errors.New("wait time out")
				}
			}
		}
		return nil
	})
}

// GetUUID .
func (g *Guest) GetUUID() (uuid string, err error) {
	err = g.botOperate(func(bot Bot) error {
		uuid, err = bot.GetUUID()
		return err
	})
	return
}

func (g *Guest) botOperate(fn func(bot Bot) error, skipLock ...bool) error {
	var bot, err = g.newBot(g)
	if err != nil {
		return errors.Wrap(err, "")
	}

	defer bot.Close()

	if !(len(skipLock) > 0 && skipLock[0]) {
		if err := bot.Trylock(); err != nil {
			return errors.Wrap(err, "")
		}
		defer bot.Unlock()
	}
	return fn(bot)
}

// CacheImage downloads the image from hub.
func (g *Guest) CacheImage(_ sync.Locker) error {
	// not implemented
	return nil
}

func (g *Guest) sysVolume() (vol volume.Volume, err error) {
	g.rangeVolumes(func(_ int, v volume.Volume) bool {
		if v.IsSys() {
			vol = v
			return false
		}
		return true
	})

	if vol == nil {
		err = errors.Wrapf(terrors.ErrSysVolumeNotExists, g.ID)
	}

	return
}

func (g *Guest) rangeVolumes(fn func(int, volume.Volume) bool) {
	for i, vol := range g.Vols {
		if !fn(i, vol) {
			return
		}
	}
}

// Distro .
func (g *Guest) Distro() string {
	return g.Img.OS.Distrib
}

// AttachConsole .
func (g *Guest) AttachConsole(ctx context.Context, serverStream io.ReadWriteCloser, flags types.OpenConsoleFlags) error {
	return g.botOperate(func(bot Bot) error {
		// epoller should bind with deamon's lifecycle but just init/destroy here for simplicity
		console, err := bot.OpenConsole(ctx, flags)
		if err != nil {
			return errors.Wrap(err, "")
		}

		done1 := make(chan struct{})
		done2 := make(chan struct{})

		// pty -> user
		go func() {
			defer func() {
				close(done1)
				log.Infof(ctx, "[guest.AttachConsole] copy console stream goroutine exited")
			}()
			console.To(ctx, serverStream) //nolint
		}()

		// user -> pty
		go func() {
			defer func() {
				close(done2)
				log.Infof(ctx, "[guest.AttachConsole] copy server stream goroutine exited")
			}()

			initCmds := append([]byte(strings.Join(flags.Commands, " ")), '\r')
			reader := io.MultiReader(bytes.NewBuffer(initCmds), serverStream)
			console.From(ctx, reader) //nolint
		}()

		// either copy goroutine exit
		select {
		case <-done1:
		case <-done2:
		case <-ctx.Done():
			log.Debugf(ctx, "[guest.AttachConsole] context done")
		}
		console.Close()
		<-done1
		<-done2
		log.Infof(ctx, "[guest.AttachConsole] exit.")
		// g.ExecuteCommand(context.Background(), []string{"yaexec", "kill"}) //nolint
		// log.Infof(ctx, "[guest.AttachConsole] yaexec completes: %v", commands)
		return nil
	})
}

// ResizeConsoleWindow .
func (g *Guest) ResizeConsoleWindow(ctx context.Context, height, width uint) (err error) { //nolint
	// TODO better way to resize console window size
	return nil
	// return g.botOperate(func(bot Bot) error {
	// 	resizeCmd := fmt.Sprintf("yaexec resize -r %d -c %d", height, width)
	// 	output, code, _, err := g.ExecuteCommand(ctx, strings.Split(resizeCmd, " "))
	// 	if code != 0 || err != nil {
	// 		log.Errorf("[guest.ResizeConsoleWindow] resize failed: %v, %v", output, err)
	// 	}
	// 	return err
	// }, true)
}

// Cat .
func (g *Guest) Cat(ctx context.Context, path string, dest io.Writer) error {
	return g.botOperate(func(bot Bot) error {
		src, err := bot.OpenFile(ctx, path, "r")
		if err != nil {
			return errors.Wrap(err, "")
		}

		defer src.Close(ctx)

		_, err = src.CopyTo(ctx, dest)

		return err
	})
}

// Log .
func (g *Guest) Log(ctx context.Context, n int, logPath string, dest io.WriteCloser) error {
	return g.botOperate(func(bot Bot) error {
		if n == 0 {
			return nil
		}
		switch g.Status {
		case meta.StatusRunning:
			return g.logRunning(ctx, bot, n, logPath, dest)
		case meta.StatusStopped:
			gfx, err := g.getGfx(logPath)
			if err != nil {
				return err
			}
			defer gfx.Close()
			return g.logStopped(n, logPath, dest, gfx)
		default:
			return errors.Wrapf(terrors.ErrNotValidLogStatus, "guest is %s", g.Status)
		}
	})
}

// CopyToGuest .
func (g *Guest) CopyToGuest(ctx context.Context, dest string, content chan []byte, overrideFolder bool) error {
	return g.botOperate(func(bot Bot) error {
		switch g.Status {
		case meta.StatusRunning:
			return g.copyToGuestRunning(ctx, dest, content, bot, overrideFolder)
		case meta.StatusStopped:
			fallthrough
		case meta.StatusCreating:
			gfx, err := g.getGfx(dest)
			if err != nil {
				return errors.Wrap(err, "")
			}
			defer gfx.Close()
			return g.copyToGuestNotRunning(dest, content, overrideFolder, gfx)
		default:
			return errors.Wrapf(terrors.ErrNotValidCopyStatus, "guest is %s", g.Status)
		}
	})
}

// ExecuteCommand .
func (g *Guest) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	err = g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Wrap(err, "")
		case st != libvirt.DomainRunning:
			return errors.Wrapf(terrors.ErrExecOnNonRunningGuest, g.ID)
		}

		output, exitCode, pid, err = bot.ExecuteCommand(ctx, commands)
		return err
	}, true)
	return
}

// nextVolumeName .
// 这里不能通过guest的vols长度来生成名字，原因如下：
// vda, vdb, vdc, 如果detach vdb, 那么这时候长度为2, 在生成名字就是vdc, 那么就冲突了
func (g *Guest) nextVolumeName() string {
	seenDev := make(map[string]bool)
	g.rangeVolumes(func(_ int, vol volume.Volume) bool {
		seenDev[vol.GetDevice()] = true
		return true
	})

	for idx := 0; idx < 26; idx++ {
		dev := base.GetDeviceName(idx)
		if !seenDev[dev] {
			return dev
		}
	}
	return ""
}
