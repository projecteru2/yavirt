package manager

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt"
	"github.com/projecteru2/yavirt/virt/guest"
	"github.com/projecteru2/yavirt/virt/types"
)

// Manageable wraps a group of methods.
type Manageable interface {
	Controllable
	Executable
	Imageable
	Creatable
	Networkable
	Loadable
	Snapshotable
}

// Creatable wraps a group of methods about creation.
type Creatable interface {
	Create(ctx virt.Context, opts types.GuestCreateOption, host *model.Host, vols []*model.Volume) (vg *guest.Guest, err error)
}

// Networkable wraps a group of networking methods.
type Networkable interface {
	ConnectExtraNetwork(ctx virt.Context, id, network, ipv4 string) (string, error)
	DisconnectExtraNetwork(ctx virt.Context, id, network string) error
}

// Executable wraps a group of executable methods.
type Executable interface {
	AttachConsole(ctx virt.Context, id string, stream io.ReadWriteCloser, flags types.OpenConsoleFlags) (err error)
	ResizeConsoleWindow(ctx virt.Context, id string, height, width uint) (err error)
	ExecuteCommand(ctx virt.Context, id string, commands []string) (output []byte, exitCode, pid int, err error)
	Cat(ctx virt.Context, id, path string, dest io.WriteCloser) error
	CopyToGuest(ctx virt.Context, id, dest string, content chan []byte, override bool) error
	Log(ctx virt.Context, id, logPath string, n int, dest io.WriteCloser) error
}

// Controllable wraps a group of controlling methods.
type Controllable interface {
	Resize(ctx virt.Context, id string, cpu int, mem int64, vols map[string]int64) error
	Start(ctx virt.Context, id string) error
	Suspend(ctx virt.Context, id string) error
	Resume(ctx virt.Context, id string) error
	Stop(ctx virt.Context, id string, force bool) error
	Destroy(ctx virt.Context, id string, force bool) (<-chan error, error)
	Wait(ctx virt.Context, id string, block bool) (msg string, code int, err error)
}

// Loadable wraps a group of loadable methods.
type Loadable interface {
	Load(ctx virt.Context, id string) (*guest.Guest, error)
	LoadUUID(ctx virt.Context, id string) (string, error)
}

var imageMutex sync.Mutex

// Imageable wraps a group of methods about images.
type Imageable interface {
	Capture(ctx virt.Context, guestID, user, name string, overridden bool) (*model.UserImage, error)
	RemoveImage(ctx virt.Context, imageName, user string, force, prune bool) ([]string, error)
	ListImage(ctx virt.Context, filter string) ([]model.Image, error)
	DigestImage(ctx virt.Context, name string, local bool) ([]string, error)
}

// Snapshotable wraps a group a methods about snapshots.
type Snapshotable interface {
	ListSnapshot(ctx virt.Context, guestID, volID string) (map[*model.Volume]model.Snapshots, error)
	CreateSnapshot(ctx virt.Context, id, volID string) error
	CommitSnapshot(ctx virt.Context, id, volID, snapID string) error
	CommitSnapshotByDay(ctx virt.Context, id, volID string, day int) error
	RestoreSnapshot(ctx virt.Context, id, volID, snapID string) error
}

// Manager implements the Manageable interface.
type Manager struct {
	serializer *serializer
}

// New initializes a new Manager instance.
func New() Manager {
	return Manager{
		serializer: newSerializer(),
	}
}

// Destroy destroys a guest.
func (m Manager) Destroy(ctx virt.Context, id string, force bool) (<-chan error, error) {
	var done <-chan error
	err := m.ctrl(ctx, id, destroyOp, func(g *guest.Guest) (de error) {
		done, de = g.Destroy(force)
		return
	}, nil)
	return done, err
}

// Stop stops a guest.
func (m Manager) Stop(ctx virt.Context, id string, force bool) error {
	return m.ctrl(ctx, id, shutOp, func(g *guest.Guest) error {
		return g.Stop(force)
	}, nil)
}

// Start boots a guest.
func (m Manager) Start(ctx virt.Context, id string) error {
	return m.ctrl(ctx, id, bootOp, func(g *guest.Guest) error {
		if err := g.Start(); err != nil {
			return err
		}

		if g.LambdaOption != nil && !g.LambdaStdin {
			output, exitCode, pid, err := g.ExecuteCommand(ctx, g.LambdaOption.Cmd)
			if err != nil {
				return err
			}
			g.LambdaOption.CmdOutput = output
			g.LambdaOption.ExitCode = exitCode
			g.LambdaOption.Pid = pid

			if err = g.Save(); err != nil {
				return err
			}
		}

		return nil
	}, nil)
}

// Wait for a guest.
func (m Manager) Wait(ctx virt.Context, id string, block bool) (msg string, code int, err error) {
	err = m.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		if err = g.Wait(model.StatusStopped, block); err != nil {
			return err
		}

		if g.LambdaOption != nil {
			msg = string(g.LambdaOption.CmdOutput)
			code = g.LambdaOption.ExitCode
		}

		return nil
	}, nil)
	return msg, code, err
}

// Suspend suspends a guest.
func (m Manager) Suspend(ctx virt.Context, id string) error {
	return m.ctrl(ctx, id, bootOp, func(g *guest.Guest) error {
		return g.Suspend()
	}, nil)
}

// Resume resumes a suspended guest.
func (m Manager) Resume(ctx virt.Context, id string) error {
	return m.ctrl(ctx, id, bootOp, func(g *guest.Guest) error {
		return g.Resume()
	}, nil)
}

// Resize re-allocates spec or volumes.
func (m Manager) Resize(ctx virt.Context, id string, cpu int, mem int64, vols map[string]int64) error {
	return m.ctrl(ctx, id, resizeOp, func(g *guest.Guest) error {
		return g.Resize(cpu, mem, vols)
	}, nil)
}

// List snapshots of volume.
func (m Manager) ListSnapshot(ctx virt.Context, guestID, volID string) (map[*model.Volume]model.Snapshots, error) {
	g, err := m.Load(ctx, guestID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return g.ListSnapshot(volID)
}

// Create snapshot of volume with volID
func (m Manager) CreateSnapshot(ctx virt.Context, id, volID string) error {
	return m.ctrl(ctx, id, createSnapshotOp, func(g *guest.Guest) error {
		suspended := false
		stopped := false
		if g.Status == model.StatusRunning {
			if err := g.Suspend(); err != nil {
				return err
			}
			suspended = true
		}

		if err := g.CreateSnapshot(volID); err != nil {
			return err
		}

		if err := g.CheckVolume(volID); err != nil {

			if suspended {
				if err := g.Stop(true); err != nil {
					return err
				}
				suspended = false
				stopped = true
			}

			if err := g.RepairVolume(volID); err != nil {
				return err
			}
		}

		if suspended {
			return g.Resume()
		} else if stopped {
			return g.Start()
		}
		return nil
	}, nil)
}

// Commit snapshot (with snapID) to the root backing file
func (m Manager) CommitSnapshot(ctx virt.Context, id, volID, snapID string) error {
	return m.ctrl(ctx, id, commitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == model.StatusRunning {
			if err := g.Stop(true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshot(volID, snapID); err != nil {
			return err
		}

		if stopped {
			return g.Start()
		}
		return nil
	}, nil)
}

// Commit snapshot that created `day` days before
func (m Manager) CommitSnapshotByDay(ctx virt.Context, id, volID string, day int) error {
	return m.ctrl(ctx, id, commitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == model.StatusRunning {
			if err := g.Stop(true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshotByDay(volID, day); err != nil {
			return err
		}

		if stopped {
			return g.Start()
		}
		return nil
	}, nil)
}

// Restore volume to snapshot with snapID
func (m Manager) RestoreSnapshot(ctx virt.Context, id, volID, snapID string) error {
	return m.ctrl(ctx, id, restoreSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == model.StatusRunning {
			if err := g.Stop(true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.RestoreSnapshot(volID, snapID); err != nil {
			return err
		}

		if stopped {
			return g.Start()
		}
		return nil
	}, nil)
}

// Capture captures an image from a guest.
func (m Manager) Capture(ctx virt.Context, guestID, user, name string, overridden bool) (*model.UserImage, error) {
	g, err := m.Load(ctx, guestID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	uImg, err := g.Capture(user, name, overridden)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return uImg, nil
}

// RemoveImage removes a local image.
func (m Manager) RemoveImage(ctx virt.Context, imageName, user string, force, prune bool) ([]string, error) {
	img, err := model.LoadImage(imageName, user)
	if err != nil {
		return nil, errors.Trace(err)
	}

	imageMutex.Lock()
	defer imageMutex.Unlock()

	if exists, err := model.ImageExists(img); err != nil {
		return nil, errors.Trace(err)
	} else if exists {
		if err = os.Remove(img.Filepath()); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return []string{img.GetID()}, nil
}

// ListImage .
func (m Manager) ListImage(ctx virt.Context, filter string) ([]model.Image, error) {
	imgs, err := model.ListSysImages()
	if err != nil {
		return nil, err
	}

	if len(filter) < 1 {
		return imgs, nil
	}

	images := []model.Image{}
	var regExp *regexp.Regexp
	filter = strings.ReplaceAll(filter, "*", ".*")
	if regExp, err = regexp.Compile(fmt.Sprintf("%s%s%s", "^", filter, "$")); err != nil {
		return nil, err
	}

	for _, img := range imgs {
		if regExp.MatchString(img.GetName()) {
			images = append(images, img)
		}
	}

	return images, nil
}

// DigestImage .
func (m Manager) DigestImage(ctx virt.Context, name string, local bool) ([]string, error) {
	if !local {
		// TODO: wait for image-hub implementation and calico update
		return []string{""}, nil
	}

	// If not exists return error
	// If exists return digests

	img, err := model.LoadSysImage(name)
	if err != nil {
		return nil, err
	}

	hash, err := img.UpdateHash()
	if err != nil {
		return nil, err
	}

	return []string{hash}, nil
}

// Create creates a new guest.
func (m Manager) Create(ctx virt.Context, opts types.GuestCreateOption, host *model.Host, vols []*model.Volume) (*guest.Guest, error) {
	// Creates metadata.
	g, err := model.CreateGuest(opts, host, vols)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Destroys resource and delete metadata while rolling back.
	var vg *guest.Guest
	destroy := func() {
		if vg == nil {
			return
		}

		done, err := vg.Destroy(true)
		if err != nil {
			log.ErrorStack(err)
		}

		select {
		case err := <-done:
			if err != nil {
				log.ErrorStack(err)
			}
		case <-time.After(time.Minute):
			log.ErrorStackf(errors.ErrTimeout, "destroy timeout")
		}
	}

	// Creates the resource.
	create := func(_ *guest.Guest) (interface{}, error) {
		var err error
		vg, err = m.create(ctx, g)
		return vg, err
	}

	res, err := m.doCtrl(ctx, g.ID, createOp, create, destroy)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return res.(*guest.Guest), nil
}

func (m Manager) create(ctx virt.Context, g *model.Guest) (vg *guest.Guest, err error) {
	vg = guest.New(ctx, g)
	if err := vg.CacheImage(&imageMutex); err != nil {
		return nil, errors.Trace(err)
	}

	if vg.MAC, err = util.QemuMAC(); err != nil {
		return nil, errors.Trace(err)
	}

	var rollback func() error
	if rollback, err = vg.CreateEthernet(); err != nil {
		return nil, errors.Trace(err)
	}

	if err = vg.Create(); err != nil {
		if re := rollback(); re != nil {
			err = errors.Wrap(err, re)
		}
		return nil, errors.Trace(err)
	}

	return vg, nil
}

// AttachConsole attaches to a guest's console.
func (m Manager) AttachConsole(ctx virt.Context, id string, stream io.ReadWriteCloser, flags types.OpenConsoleFlags) error {
	g, err := m.Load(ctx, id)
	if err != nil {
		return errors.Trace(err)
	}

	if g.LambdaOption != nil {
		if err = g.Wait(model.StatusRunning, false); err != nil {
			return errors.Trace(err)
		}
		flags.Commands = g.LambdaOption.Cmd
	}

	return g.AttachConsole(ctx, stream, flags)
}

// ResizeConsoleWindow resizes a console's window.
func (m Manager) ResizeConsoleWindow(ctx virt.Context, id string, height, width uint) error {
	g, err := m.Load(ctx, id)
	if err != nil {
		return errors.Trace(err)
	}
	return g.ResizeConsoleWindow(ctx, height, width)
}

type executeResult struct {
	output   []byte
	exitCode int
	pid      int
}

// ExecuteCommand executes commands.
func (m Manager) ExecuteCommand(ctx virt.Context, id string, commands []string) (content []byte, exitCode, pid int, err error) {
	exec := func(g *guest.Guest) (interface{}, error) {
		output, exitCode, pid, err := g.ExecuteCommand(ctx, commands)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &executeResult{output: output, exitCode: exitCode, pid: pid}, nil
	}

	res, err := m.doCtrl(ctx, id, miscOp, exec, nil)
	if err != nil {
		return nil, -1, -1, errors.Trace(err)
	}

	er, ok := res.(*executeResult)
	if !ok {
		return nil, -1, -1, errors.Annotatef(errors.ErrInvalidValue, "expect *executeResult but it's %v", res)
	}
	return er.output, er.exitCode, er.pid, nil
}

// Cat cats the file that in the guest.
func (m Manager) Cat(ctx virt.Context, id, path string, dest io.WriteCloser) error {
	return m.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.Cat(ctx, path, dest)
	}, nil)
}

// Log shows the log file.
func (m Manager) Log(ctx virt.Context, id, logPath string, n int, dest io.WriteCloser) error {
	return m.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		if g.LambdaOption == nil {
			return g.Log(ctx, n, logPath, dest)
		}

		defer dest.Close()
		_, err := dest.Write(g.LambdaOption.CmdOutput)
		return err
	}, nil)
}

// CopyToGuest copy file to guest
func (m Manager) CopyToGuest(ctx virt.Context, id, dest string, content chan []byte, override bool) error {
	return m.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.CopyToGuest(ctx, dest, content, override)
	}, nil)
}

// DisconnectExtraNetwork disconnects from an extra network.
func (m Manager) DisconnectExtraNetwork(ctx virt.Context, id, network string) error {
	return m.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.DisconnectExtraNetwork(network)
	}, nil)
}

// ConnectExtraNetwork connects to an extra network.
func (m Manager) ConnectExtraNetwork(ctx virt.Context, id, network, ipv4 string) (string, error) {
	var ip meta.IP

	if err := m.ctrl(ctx, id, miscOp, func(g *guest.Guest) (ce error) {
		ip, ce = g.ConnectExtraNetwork(network, ipv4)
		return ce
	}, nil); err != nil {
		return "", errors.Trace(err)
	}

	return ip.CIDR(), nil
}

type ctrlFunc func(*guest.Guest) error

func (m Manager) ctrl(ctx virt.Context, id string, op op, fn ctrlFunc, rollback rollbackFunc) error { //nolint
	_, err := m.doCtrl(ctx, id, op, func(g *guest.Guest) (interface{}, error) {
		return nil, fn(g)
	}, rollback)
	return err
}

type rollbackFunc func()
type doCtrlFunc func(*guest.Guest) (interface{}, error)

func (m Manager) doCtrl(ctx virt.Context, id string, op op, fn doCtrlFunc, rollback rollbackFunc) (interface{}, error) {
	do := func(ctx virt.Context) (interface{}, error) {
		g, err := m.Load(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return fn(g)
	}
	return m.do(ctx, id, op, do, rollback)
}

// LoadUUID read a guest's UUID.
func (m Manager) LoadUUID(ctx virt.Context, id string) (string, error) {
	g, err := m.Load(ctx, id)
	if err != nil {
		return "", errors.Trace(err)
	}
	return g.GetUUID()
}

// Load read a guest from metadata.
func (m Manager) Load(ctx virt.Context, id string) (*guest.Guest, error) {
	g, err := model.LoadGuest(id)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var vg = guest.New(ctx, g)
	if err := vg.Load(); err != nil {
		return nil, errors.Trace(err)
	}

	return vg, nil
}

type doFunc func(virt.Context) (interface{}, error)

func (m Manager) do(ctx virt.Context, id string, op op, fn doFunc, rollback rollbackFunc) (interface{}, error) {
	t := &task{
		id:  id,
		op:  op,
		do:  fn,
		ctx: ctx,
	}

	dur := config.Conf.VirtTimeout.Duration()
	timeout := time.After(dur)

	noti := m.serializer.Serialize(id, t)

	var result interface{}
	var err error

	select {
	case <-noti.done:
		result = noti.result()
		err = noti.error()
	case <-ctx.Done():
		err = ctx.Err()
	case <-timeout:
		err = errors.Annotatef(errors.ErrTimeout, "exceed %v", dur)
	}

	if err != nil {
		if rollback != nil {
			rollback()
		}
		return nil, errors.Trace(err)
	}

	return result, nil
}
