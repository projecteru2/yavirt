package server

import (
	"context"
	"fmt"
	"io"

	"github.com/projecteru2/libyavirt/types"

	"github.com/robfig/cron/v3"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/internal/virt/guest/manager"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	vlanhandler "github.com/projecteru2/yavirt/internal/vnet/handler/vlan"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Service .
type Service struct {
	Host        *models.Host
	BootGuestCh chan<- string
	caliHandler *calihandler.Handler
	guest       manager.Manageable

	pid2ExitCode   *utils.ExitCodeMap
	RecoverGuestCh chan<- string
}

// SetupYavirtdService .
func SetupYavirtdService() (*Service, error) {
	svc := &Service{guest: manager.New(), pid2ExitCode: utils.NewSyncMap()}
	return svc, svc.setup()
}

func (svc *Service) setup() error {
	hn, err := utils.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	if svc.Host, err = models.LoadHost(hn); err != nil {
		return errors.Trace(err)
	}

	if err := svc.setupCalico(); err != nil {
		return errors.Trace(err)
	}

	/*
		if err := svc.ScheduleSnapshotCreate(); err != nil {
			return errors.Trace(err)
		}
	*/

	return nil
}

// TODO: Decide time
func (svc *Service) ScheduleSnapshotCreate() error {
	c := cron.New()

	// Everyday 3am
	if _, err := c.AddFunc("0 3 * * *", svc.batchCreateSnapshot); err != nil {
		return errors.Trace(err)
	}

	// Every Sunday 1am
	if _, err := c.AddFunc("0 1 * * SUN", svc.batchCommitSnapshot); err != nil {
		return errors.Trace(err)
	}

	// Start job asynchronously
	c.Start()

	return nil
}

func (svc *Service) batchCreateSnapshot() {
	guests, err := models.GetAllGuests()
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			req := types.CreateSnapshotReq{
				ID:    g.ID,
				VolID: volID,
			}

			if err := svc.CreateSnapshot(
				virt.NewContext(context.Background(), svc.caliHandler), req,
			); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}
	}
}

func (svc *Service) batchCommitSnapshot() {
	guests, err := models.GetAllGuests()
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			if err := svc.CommitSnapshotByDay(
				virt.NewContext(context.Background(), svc.caliHandler),
				g.ID,
				volID,
				configs.Conf.SnapshotRestorableDay,
			); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}
	}
}

// VirtContext .
func (svc *Service) VirtContext(ctx context.Context) virt.Context {
	return virt.NewContext(ctx, svc.caliHandler)
}

// Ping .
func (svc *Service) Ping() map[string]string {
	return map[string]string{"version": ver.Version()}
}

// Info .
func (svc *Service) Info() types.HostInfo {
	return types.HostInfo{
		ID:      fmt.Sprintf("%d", svc.Host.ID),
		CPU:     svc.Host.CPU,
		Mem:     svc.Host.Memory,
		Storage: svc.Host.Storage,
	}
}

// GetGuest .
func (svc *Service) GetGuest(ctx virt.Context, id string) (*types.Guest, error) {
	vg, err := svc.guest.Load(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}
	return convGuestResp(vg.Guest), nil
}

// GetGuestUUID .
func (svc *Service) GetGuestUUID(ctx virt.Context, id string) (string, error) {
	uuid, err := svc.guest.LoadUUID(ctx, id)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "", err
	}
	return uuid, nil
}

// CreateGuest .
func (svc *Service) CreateGuest(ctx virt.Context, opts virtypes.GuestCreateOption) (*types.Guest, error) {
	vols := []*models.Volume{}
	for mnt, capacity := range opts.Volumes {
		vol, err := models.NewDataVolume(mnt, capacity)
		if err != nil {
			return nil, errors.Trace(err)
		}
		vols = append(vols, vol)
	}

	if opts.CPU == 0 {
		opts.CPU = utils.Min(svc.Host.CPU, configs.Conf.MaxCPU)
	}
	if opts.Mem == 0 {
		opts.Mem = utils.MinInt64(svc.Host.Memory, configs.Conf.MaxMemory)
	}

	g, err := svc.guest.Create(ctx, opts, svc.Host, vols)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return nil, err
	}

	go func() {
		svc.BootGuestCh <- g.ID
	}()

	return convGuestResp(g.Guest), nil
}

// CaptureGuest .
func (svc *Service) CaptureGuest(ctx virt.Context, req types.CaptureGuestReq) (uimg *models.UserImage, err error) {
	if uimg, err = svc.guest.Capture(ctx, req.VirtID(), req.User, req.Name, req.Overridden); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ResizeGuest .
func (svc *Service) ResizeGuest(ctx virt.Context, req types.ResizeGuestReq) (err error) {
	if err = svc.guest.Resize(ctx, req.VirtID(), req.CPU, req.Mem, req.Volumes); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ControlGuest .
func (svc *Service) ControlGuest(ctx virt.Context, id, operation string, force bool) (err error) {
	switch operation {
	case types.OpStart:
		err = svc.guest.Start(ctx, id)
	case types.OpStop:
		err = svc.guest.Stop(ctx, id, force)
	case types.OpDestroy:
		_, err = svc.guest.Destroy(ctx, id, force)
	}

	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return errors.Trace(err)
	}

	return nil
}

// ListSnapshot .
func (svc *Service) ListSnapshot(ctx virt.Context, req types.ListSnapshotReq) (snaps types.Snapshots, err error) {
	volSnap, err := svc.guest.ListSnapshot(ctx, req.ID, req.VolID)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}

	for vol, s := range volSnap {
		for _, snap := range s {
			snaps = append(snaps, &types.Snapshot{
				VolID:       vol.ID,
				VolMountDir: vol.GetMountDir(),
				SnapID:      snap.ID,
				CreatedTime: snap.CreatedTime,
			})
		}
	}

	return
}

// CreateSnapshot .
func (svc *Service) CreateSnapshot(ctx virt.Context, req types.CreateSnapshotReq) (err error) {
	if err = svc.guest.CreateSnapshot(ctx, req.ID, req.VolID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CommitSnapshot .
func (svc *Service) CommitSnapshot(ctx virt.Context, req types.CommitSnapshotReq) (err error) {
	if err = svc.guest.CommitSnapshot(ctx, req.ID, req.VolID, req.SnapID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CommitSnapshotByDay .
func (svc *Service) CommitSnapshotByDay(ctx virt.Context, id, volID string, day int) (err error) {
	if err = svc.guest.CommitSnapshotByDay(ctx, id, volID, day); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// RestoreSnapshot .
func (svc *Service) RestoreSnapshot(ctx virt.Context, req types.RestoreSnapshotReq) (err error) {
	if err = svc.guest.RestoreSnapshot(ctx, req.ID, req.VolID, req.SnapID); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ConnectNetwork .
func (svc *Service) ConnectNetwork(ctx virt.Context, id, network, ipv4 string) (cidr string, err error) {
	if cidr, err = svc.guest.ConnectExtraNetwork(ctx, id, network, ipv4); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// DisconnectNetwork .
func (svc *Service) DisconnectNetwork(ctx virt.Context, id, network string) (err error) {
	if err = svc.guest.DisconnectExtraNetwork(ctx, id, network); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// NetworkList .
func (svc *Service) NetworkList(ctx virt.Context, drivers []string) ([]*types.Network, error) {
	drv := map[string]struct{}{}
	for _, driver := range drivers {
		drv[driver] = struct{}{}
	}

	networks := []*types.Network{}
	switch svc.Host.NetworkMode {
	case vnet.NetworkCalico:
		if _, ok := drv[vnet.NetworkCalico]; svc.caliHandler == nil || !ok {
			break
		}
		for _, poolName := range svc.caliHandler.PoolNames() {
			subnet, err := svc.caliHandler.GetIPPoolCidr(ctx.Context, poolName)
			if err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
				return nil, err
			}

			networks = append(networks, &types.Network{
				Name:    poolName,
				Subnets: []string{subnet},
			})
		}
		return networks, nil
	case vnet.NetworkVlan: // vlan
		if _, ok := drv[vnet.NetworkVlan]; !ok {
			break
		}
		handler := vlanhandler.New("", svc.Host.Subnet)
		networks = append(networks, &types.Network{
			Name:    "vlan",
			Subnets: []string{handler.GetCidr()},
		})
	}

	return networks, nil
}

// AttachGuest .
func (svc *Service) AttachGuest(ctx virt.Context, id string, stream io.ReadWriteCloser, flags virtypes.OpenConsoleFlags) (err error) {
	if err = svc.guest.AttachConsole(ctx, id, stream, flags); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ResizeConsoleWindow .
func (svc *Service) ResizeConsoleWindow(ctx virt.Context, id string, height, width uint) (err error) {
	if err = svc.guest.ResizeConsoleWindow(ctx, id, height, width); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// ExecuteGuest .
func (svc *Service) ExecuteGuest(ctx virt.Context, id string, commands []string) (*types.ExecuteGuestMessage, error) {
	stdout, exitCode, pid, err := svc.guest.ExecuteCommand(ctx, id, commands)
	if err != nil {
		log.WarnStack(err)
		metrics.IncrError()
	}
	svc.pid2ExitCode.Put(id, pid, exitCode)
	return &types.ExecuteGuestMessage{
		Pid:      pid,
		Data:     stdout,
		ExitCode: exitCode,
	}, err
}

// ExecExitCode .
func (svc *Service) ExecExitCode(id string, pid int) (int, error) {
	exitCode, err := svc.pid2ExitCode.Get(id, pid)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return 0, err
	}
	return exitCode, nil
}

// Cat .
func (svc *Service) Cat(ctx virt.Context, id, path string, dest io.WriteCloser) (err error) {
	if err = svc.guest.Cat(ctx, id, path, dest); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// CopyToGuest .
func (svc *Service) CopyToGuest(ctx virt.Context, id, dest string, content chan []byte, override bool) (err error) {
	if err = svc.guest.CopyToGuest(ctx, id, dest, content, override); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// Log .
func (svc *Service) Log(ctx virt.Context, id, logPath string, n int, dest io.WriteCloser) (err error) {
	if err = svc.guest.Log(ctx, id, logPath, n, dest); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// Wait .
func (svc *Service) Wait(ctx virt.Context, id string, block bool) (msg string, code int, err error) {
	err = svc.guest.Stop(ctx, id, !block)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "stop error", -1, err
	}
	if msg, code, err = svc.guest.Wait(ctx, id, block); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

func (svc *Service) PushImage(_ virt.Context, _, _ string) (err error) {
	// todo
	return
}

func (svc *Service) RemoveImage(ctx virt.Context, imageName, user string, force, prune bool) (removed []string, err error) {
	if removed, err = svc.guest.RemoveImage(ctx, imageName, user, force, prune); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

func (svc *Service) ListImage(ctx virt.Context, filter string) ([]types.SysImage, error) {
	imgs, err := svc.guest.ListImage(ctx, filter)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}

	images := []types.SysImage{}
	for _, img := range imgs {
		images = append(images, types.SysImage{
			Name:   img.GetName(),
			User:   img.GetUser(),
			Distro: img.GetDistro(),
			Id:     img.GetID(),
			Type:   img.GetType(),
		})
	}

	return images, err
}

func (svc *Service) PullImage(virt.Context, string, bool) (msg string, err error) {
	// todo
	return
}

func (svc *Service) DigestImage(ctx virt.Context, imageName string, local bool) (digest []string, err error) {
	if digest, err = svc.guest.DigestImage(ctx, imageName, local); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}
