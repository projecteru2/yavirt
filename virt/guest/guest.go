package guest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt"
	"github.com/projecteru2/yavirt/virt/types"
	"github.com/projecteru2/yavirt/virt/volume"
)

// Guest .
type Guest struct {
	*model.Guest

	ctx virt.Context

	newBot func(*Guest) (Bot, error)
}

// New initializes a new Guest.
func New(ctx virt.Context, g *model.Guest) *Guest {
	return &Guest{
		Guest:  g,
		ctx:    ctx,
		newBot: newVirtGuest,
	}
}

// Load .
func (g *Guest) Load() error {
	host, err := model.LoadHost(g.HostName)
	if err != nil {
		return errors.Trace(err)
	}

	hand, err := g.NetworkHandler(host)
	if err != nil {
		return errors.Trace(err)
	}

	if err := g.Guest.Load(host, hand); err != nil {
		return errors.Trace(err)
	}

	return g.loadExtraNetworks()
}

// SyncState .
func (g *Guest) SyncState() error {
	switch g.Status {
	case model.StatusDestroying:
		return g.ProcessDestroy()

	case model.StatusStopping:
		return g.stop(true)

	case model.StatusRunning:
		fallthrough
	case model.StatusStarting:
		return g.start()

	case model.StatusCreating:
		return g.create()

	default:
		// nothing to do
		return nil
	}
}

// Start .
func (g *Guest) Start() error {
	return util.Invoke([]func() error{
		g.ForwardStarting,
		g.start,
	})
}

func (g *Guest) start() error {
	return g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Trace(err)
		case st == libvirt.DomainRunning:
			return nil
		}

		return util.Invoke([]func() error{
			bot.Boot,
			g.joinEthernet,
			g.ForwardRunning,
		})
	})
}

// Resize .
func (g *Guest) Resize(cpu int, mem int64, mntCaps map[string]int64) error {
	// Only checking, without touch metadata
	// due to we wanna keep further booting successfully all the time.
	if !g.CheckForwardStatus(model.StatusResizing) {
		return errors.Annotatef(errors.ErrForwardStatus, "only stopped/running guest can be resized, but it's %s", g.Status)
	}

	// Actually, mntCaps from ERU will include completed volumes,
	// even those volumes aren't affected.
	if len(mntCaps) > 0 {
		// Just amplifies those original volumes.
		if err := g.amplifyOrigVols(mntCaps); err != nil {
			return errors.Trace(err)
		}
		// Attaches new extra volumes.
		if err := g.attachVols(mntCaps); err != nil {
			return errors.Trace(err)
		}
	}

	if cpu == g.CPU && mem == g.Memory {
		return nil
	}

	return g.resizeSpec(cpu, mem)
}

func (g *Guest) amplifyOrigVols(mntCaps map[string]int64) error {
	newCapMods := map[string]*model.Volume{}
	for mnt, cap := range mntCaps {
		mod, err := model.NewDataVolume(mnt, cap)
		if err != nil {
			return errors.Trace(err)
		}
		newCapMods[mod.MountDir] = mod
	}

	var err error
	g.rangeVolumes(func(sn int, vol volume.Virt) bool {
		newCapMod, affected := newCapMods[vol.Model().MountDir]
		if !affected {
			return true
		}

		var delta int64
		switch delta = newCapMod.Capacity - vol.Model().Capacity; {
		case delta < 0:
			err = errors.Annotatef(errors.ErrCannotShrinkVolume, "mount dir: %s", newCapMod.MountDir)
			return false
		case delta == 0: // nothing changed
			return true
		}

		err = g.botOperate(func(bot Bot) error {
			return bot.AmplifyVolume(vol, delta, vol.Model().GetDevicePathBySerialNumber(sn))
		})
		return err == nil
	})

	return err
}

func (g *Guest) attachVols(mntCaps map[string]int64) error {
	for mnt, cap := range mntCaps {
		volmod, err := model.NewDataVolume(mnt, cap)
		switch {
		case err != nil:
			return errors.Trace(err)
		case g.Vols.Exists(volmod.MountDir):
			continue
		}

		volmod.GuestID = g.ID
		volmod.Status = g.Status
		volmod.GenerateID()
		if err := g.attachVol(volmod); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (g *Guest) attachVol(volmod *model.Volume) (err error) {
	devName := g.nextVolumeName()

	if err = g.AppendVols(volmod); err != nil {
		return errors.Trace(err)
	}

	var rollback func()
	defer func() {
		if err != nil {
			if rollback != nil {
				rollback()
			}
			g.RemoveVol(volmod.ID)
		}
	}()

	if err = g.botOperate(func(bot Bot) (ae error) {
		rollback, ae = bot.AttachVolume(volmod, devName)
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
		return errors.Trace(err)
	}

	return g.Guest.Resize(cpu, mem)
}

// ListSnapshot If volID == "", list snapshots of all vols. Else will find vol with matching volID.
func (g *Guest) ListSnapshot(volID string) (map[*model.Volume]model.Snapshots, error) {
	volSnap := make(map[*model.Volume]model.Snapshots)

	matched := false
	for _, v := range g.Vols {
		if v.ID == volID || volID == "" {
			volSnap[v] = v.Snaps
			matched = true
		}
	}

	if !matched {
		return nil, errors.Annotatef(errors.ErrInvalidValue, "volID %s not exists", volID)
	}

	return volSnap, nil
}

// CheckVolume .
func (g *Guest) CheckVolume(volID string) error {
	if g.Status != model.StatusStopped && g.Status != model.StatusPaused {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only paused/stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CheckVolume(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// RepairVolume .
func (g *Guest) RepairVolume(volID string) error {
	if g.Status != model.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform volume check, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RepairVolume(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CreateSnapshot .
func (g *Guest) CreateSnapshot(volID string) error {
	if g.Status != model.StatusStopped && g.Status != model.StatusPaused {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only paused/stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CreateSnapshot(vol)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshot(volID string, snapID string) error {
	if g.Status != model.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshot(vol, snapID)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// CommitSnapshot .
func (g *Guest) CommitSnapshotByDay(volID string, day int) error {
	if g.Status != model.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.CommitSnapshotByDay(vol, day)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// RestoreSnapshot .
func (g *Guest) RestoreSnapshot(volID string, snapID string) error {
	if g.Status != model.StatusStopped {
		return errors.Annotatef(errors.ErrForwardStatus,
			"only stopped guest can be perform snapshot operation, but it's %s", g.Status)
	}

	vol, err := g.Vols.Find(volID)
	if err != nil {
		return err
	}

	if err := g.botOperate(func(bot Bot) error {
		return bot.RestoreSnapshot(vol, snapID)
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Capture .
func (g *Guest) Capture(user, name string, overridden bool) (uimg *model.UserImage, err error) {
	var orig *model.UserImage
	if overridden {
		if orig, err = model.LoadUserImage(user, name); err != nil {
			return
		}
	}

	if err = g.ForwardCapturing(); err != nil {
		return
	}

	if err = g.botOperate(func(bot Bot) error {
		var ce error
		if uimg, ce = bot.Capture(user, name); ce != nil {
			return errors.Trace(ce)
		}
		return g.ForwardCaptured()
	}); err != nil {
		return
	}

	if err = g.ForwardStopped(false); err != nil {
		return
	}

	if overridden {
		orig.Distro = uimg.Distro
		orig.Size = uimg.Size
		err = orig.Save()
	} else {
		err = uimg.Create()
	}

	return uimg, err
}

// Migrate .
func (g *Guest) Migrate() error {
	return util.Invoke([]func() error{
		g.ForwardMigrating,
		g.migrate,
	})
}

func (g *Guest) migrate() error {
	return g.botOperate(func(bot Bot) error {
		return bot.Migrate()
	})
}

// Create .
func (g *Guest) Create() error {
	return util.Invoke([]func() error{
		g.ForwardCreating,
		g.create,
	})
}

func (g *Guest) create() error {
	return g.botOperate(func(bot Bot) error {
		return util.Invoke([]func() error{
			bot.Create,
		})
	})
}

// Stop .
func (g *Guest) Stop(force bool) error {
	if err := g.ForwardStopping(); !force && err != nil {
		return errors.Trace(err)
	}
	return g.stop(force)
}

func (g *Guest) stop(force bool) error {
	return g.botOperate(func(bot Bot) error {
		if err := bot.Shutdown(force); err != nil {
			return errors.Trace(err)
		}
		return g.ForwardStopped(force)
	})
}

// Suspend .
func (g *Guest) Suspend() error {
	return util.Invoke([]func() error{
		g.ForwardPausing,
		g.suspend,
	})
}

func (g *Guest) suspend() error {
	return g.botOperate(func(bot Bot) error {
		return util.Invoke([]func() error{
			bot.Suspend,
			g.ForwardPaused,
		})
	})
}

// Resume .
func (g *Guest) Resume() error {
	return util.Invoke([]func() error{
		g.ForwardResuming,
		g.resume,
	})
}

func (g *Guest) resume() error {
	return g.botOperate(func(bot Bot) error {
		return util.Invoke([]func() error{
			bot.Resume,
			g.ForwardRunning,
		})
	})
}

// Destroy .
func (g *Guest) Destroy(force bool) (<-chan error, error) {
	if force {
		if err := g.stop(true); err != nil && !errors.IsDomainNotExistsErr(err) {
			return nil, errors.Trace(err)
		}
	}

	if err := g.ForwardDestroying(force); err != nil {
		return nil, errors.Trace(err)
	}
	log.Infof("[guest.Destroy] set state of guest %s to destroying", g.ID)

	done := make(chan error, 1)

	// will return immediately as the destroy request has been accepted.
	// the detail work will be processed asynchronously
	go func() {
		err := g.ProcessDestroy()
		if err != nil {
			log.ErrorStackf(err, "destroy guest %s failed", g.ID)
			//TODO: move to recovery list
		}
		done <- err
	}()

	return done, nil
}

func (g *Guest) ProcessDestroy() error {
	log.Infof("[guest.destroy] begin to destroy guest %s ", g.ID)
	return g.botOperate(func(bot Bot) error {
		if err := g.DeleteNetwork(); err != nil {
			return errors.Trace(err)
		}

		if err := bot.Undefine(); err != nil {
			return errors.Trace(err)
		}

		return g.Delete(false)
	})
}

const waitRetries = 30 // 30 second
// Wait .
func (g *Guest) Wait(toStatus string, block bool) error {
	if !g.CheckForwardStatus(toStatus) {
		return errors.ErrForwardStatus
	}
	return g.botOperate(func(bot Bot) error {
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
		return errors.Trace(err)
	}

	defer bot.Close()

	if !(len(skipLock) > 0 && skipLock[0]) {
		if err := bot.Trylock(); err != nil {
			return errors.Trace(err)
		}
		defer bot.Unlock()
	}
	return fn(bot)
}

// CacheImage downloads the image from hub.
func (g *Guest) CacheImage(lock sync.Locker) error {
	// not implemented
	return nil
}

func (g *Guest) sysVolume() (vol volume.Virt, err error) {
	g.rangeVolumes(func(_ int, v volume.Virt) bool {
		if v.IsSys() {
			vol = v
			return false
		}
		return true
	})

	if vol == nil {
		err = errors.Annotatef(errors.ErrSysVolumeNotExists, g.ID)
	}

	return
}

func (g *Guest) rangeVolumes(fn func(int, volume.Virt) bool) {
	for i, volmod := range g.Vols {
		if !fn(i, volume.New(volmod)) {
			return
		}
	}
}

// Distro .
func (g *Guest) Distro() string {
	return g.Img.GetDistro()
}

// AttachConsole .
func (g *Guest) AttachConsole(ctx context.Context, serverStream io.ReadWriteCloser, flags types.OpenConsoleFlags) error {
	return g.botOperate(func(bot Bot) error {
		// epoller should bind with deamon's lifecycle but just init/destroy here for simplicity
		epoller := GetCurrentEpoller()
		if epoller == nil {
			return errors.New("Epoller is not initialized")
		}
		g.ExecuteCommand(ctx, []string{"yaexec", "kill"}) //nolint // to grapple with yavirt collapsed with yaexec alive
		console, err := bot.OpenConsole(ctx, types.NewOpenConsoleFlags(flags.Force, flags.Safe))
		if err != nil {
			return errors.Trace(err)
		}
		epollConsole, err := epoller.Add(console)
		if err != nil {
			return errors.Trace(err)
		}
		log.Infof("[guest.AttachConsole] console opened")

		commands := []string{"yaexec", "exec", "--"}
		commands = append(commands, flags.Commands...)
		g.ExecuteCommand(ctx, commands) //nolint
		log.Infof("[guest.AttachConsole] yaexec executing: %v", commands)

		copyDone := make(chan struct{})
		ctx, cancel := context.WithCancel(ctx)
		types.ConsoleStateManager.MarkConsoleOpen(ctx, g.ID)

		// pty -> user
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer func() {
				log.Infof("[guest.AttachConsole] copy console stream to server stream complete")
				types.ConsoleStateManager.MarkConsoleClose(ctx, g.ID)
				select {
				case copyDone <- struct{}{}:
				case <-ctx.Done():
				}
				wg.Done()
				log.Infof("[guest.AttachConsole] copy console stream goroutine exited")
			}()
			util.CopyIO(ctx, serverStream, epollConsole) //nolint
		}()

		// user -> pty
		wg.Add(1)
		go func() {
			defer func() {
				log.Infof("[guest.AttachConsole] copy server stream to console stream complete")
				select {
				case copyDone <- struct{}{}:
				case <-ctx.Done():
				}
				wg.Done()
				log.Infof("[guest.AttachConsole] copy server stream goroutine exited")
			}()
			util.CopyIO(ctx, epollConsole, serverStream) //nolint
		}()

		// close MPSC chan
		go func() {
			wg.Wait()
			close(copyDone)
			log.Infof("[guest.AttachConsole] copy closed")
		}()

		// either copy goroutine exit
		select {
		case <-copyDone:
		case <-ctx.Done():
		}
		cancel()
		// remove from epoller and shutdown console
		if err := epollConsole.Shutdown(epoller.CloseConsole); err != nil {
			log.Errorf("[guest.AttachConsole] failed to shutdown epoll console")
		}
		if err := epollConsole.Close(); err != nil {
			log.Errorf("[guest.AttachConsole] failed to close epoll console")
		}
		g.ExecuteCommand(context.Background(), []string{"yaexec", "kill"}) //nolint
		log.Infof("[guest.AttachConsole] yaexec completes: %v", commands)
		return nil
	})
}

// ResizeConsoleWindow .
func (g *Guest) ResizeConsoleWindow(ctx context.Context, height, width uint) (err error) {
	return g.botOperate(func(bot Bot) error {
		types.ConsoleStateManager.WaitUntilConsoleOpen(ctx, g.ID)
		resizeCmd := fmt.Sprintf("yaexec resize -r %d -c %d", height, width)
		output, code, _, err := g.ExecuteCommand(ctx, strings.Split(resizeCmd, " "))
		if code != 0 || err != nil {
			log.Errorf("[guest.ResizeConsoleWindow] resize failed: %v, %v", output, err)
		}
		return err
	}, true)
}

// Cat .
func (g *Guest) Cat(ctx context.Context, path string, dest io.WriteCloser) error {
	return g.botOperate(func(bot Bot) error {
		src, err := bot.OpenFile(path, "r")
		if err != nil {
			return errors.Trace(err)
		}

		defer src.Close()

		_, err = util.CopyIO(ctx, dest, src)

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
		case model.StatusRunning:
			return g.logRunning(ctx, bot, n, logPath, dest)
		case model.StatusStopped:
			gfx, err := g.getGfx(logPath)
			if err != nil {
				return err
			}
			defer gfx.Close()
			return g.logStopped(n, logPath, dest, gfx)
		default:
			return errors.Annotatef(errors.ErrNotValidLogStatus, "guest is %s", g.Status)
		}
	})
}

// CopyToGuest .
func (g *Guest) CopyToGuest(ctx context.Context, dest string, content chan []byte, overrideFolder bool) error {
	return g.botOperate(func(bot Bot) error {
		switch g.Status {
		case model.StatusRunning:
			return g.copyToGuestRunning(ctx, dest, content, bot, overrideFolder)
		case model.StatusStopped:
			fallthrough
		case model.StatusCreating:
			gfx, err := g.getGfx(dest)
			if err != nil {
				return errors.Trace(err)
			}
			defer gfx.Close()
			return g.copyToGuestNotRunning(dest, content, overrideFolder, gfx)
		default:
			return errors.Annotatef(errors.ErrNotValidCopyStatus, "guest is %s", g.Status)
		}
	})
}

// ExecuteCommand .
func (g *Guest) ExecuteCommand(ctx context.Context, commands []string) (output []byte, exitCode, pid int, err error) {
	err = g.botOperate(func(bot Bot) error {
		switch st, err := bot.GetState(); {
		case err != nil:
			return errors.Trace(err)
		case st != libvirt.DomainRunning:
			return errors.Annotatef(errors.ErrExecOnNonRunningGuest, g.ID)
		}

		output, exitCode, pid, err = bot.ExecuteCommand(ctx, commands)
		return err
	}, true)
	return
}

// nextVolumeName .
func (g *Guest) nextVolumeName() string {
	return model.GetDeviceName(g.Vols.Len())
}
