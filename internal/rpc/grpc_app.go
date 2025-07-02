package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	pb "github.com/projecteru2/libyavirt/grpc/gen"
	"github.com/projecteru2/libyavirt/types"
	"github.com/samber/lo"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/service"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/utils"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
)

// GRPCYavirtd .
type GRPCYavirtd struct {
	service service.Service
}

// Ping .
func (y *GRPCYavirtd) Ping(_ context.Context, _ *pb.Empty) (*pb.PingMessage, error) {
	pang := y.service.Ping()
	return &pb.PingMessage{Version: pang["version"]}, nil
}

// GetInfo .
func (y *GRPCYavirtd) GetInfo(ctx context.Context, _ *pb.Empty) (*pb.InfoMessage, error) {
	if configs.Conf.Log.Verbose {
		log.Debug(ctx, "[grpcserver] get host info")
	}
	info, err := y.service.Info()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &pb.InfoMessage{
		Id:        info.ID,
		Cpu:       int64(info.CPU),
		Memory:    info.Mem,
		Storage:   info.Storage,
		Resources: info.Resources,
	}, nil
}

// GetGuest .
func (y *GRPCYavirtd) GetGuest(ctx context.Context, opts *pb.GetGuestOptions) (*pb.GetGuestMessage, error) {
	log.WithFunc("GRPCYavirtd.GetGuest").Infof(ctx, "get guest: %s", opts.Id)
	guestReq := types.GuestReq{ID: opts.Id}
	guest, err := y.service.GetGuest(ctx, guestReq.VirtID())
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &pb.GetGuestMessage{
		Id:            types.EruID(guest.ID),
		Status:        guest.Status,
		TransitStatus: guest.TransitStatus,
		CreateTime:    guest.CreateTime,
		TransitTime:   guest.TransitTime,
		UpdateTime:    guest.UpdateTime,
		Cpu:           int64(guest.CPU),
		Memory:        guest.Mem,
		Storage:       guest.Storage,
		ImageId:       guest.ImageID,
		ImageName:     guest.ImageName,
		Networks:      guest.Networks,
		Ips:           guest.IPs,
		Labels:        guest.Labels,
		Hostname:      guest.Hostname,
		Running:       guest.Running,
	}, nil
}

// GetGuestIDList gets all local vms' domain names regardless of their metadata validility.
func (y *GRPCYavirtd) GetGuestIDList(ctx context.Context, _ *pb.GetGuestIDListOptions) (*pb.GetGuestIDListMessage, error) {
	log.WithFunc("GRPCYavirtd.GetGuestIDList").Info(ctx, "get guest id list")
	ids, err := y.service.GetGuestIDList(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	eruIDs := lo.Map(ids, func(id string, _ int) string {
		return types.EruID(id)
	})
	return &pb.GetGuestIDListMessage{Ids: eruIDs}, nil
}

// Events
func (y *GRPCYavirtd) Events(_ *pb.EventsOptions, server pb.YavirtdRPC_EventsServer) error {
	ctx := server.Context()

	log.Info(ctx, "[grpcserver] events method calling")
	defer log.Info(ctx, "[grpcserver] events method completed")

	watcher, err := y.service.WatchGuestEvents(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer log.Info(ctx, "[grpcserver] events goroutine has done")
		defer wg.Done()
		defer watcher.Stop()

		for {
			select {
			case event := <-watcher.Events():
				if err := server.Send(parseEvent(event)); err != nil {
					log.Error(ctx, err)
					return
				}

			case <-watcher.Done():
				// The watcher already has been stopped.
				log.Info(ctx, "[grpcserver] watcher has done")
				return

			case <-ctx.Done():
				log.Info(ctx, "[grpcserver] ctx done")
				return
			}
		}
	}()

	return nil
}

func parseEvent(event intertypes.Event) *pb.EventMessage {
	return &pb.EventMessage{
		Id:       types.EruID(event.ID),
		Type:     event.Type,
		Action:   string(event.Op),
		TimeNano: event.Time.UnixNano(),
	}
}

// GetGuestUUID .
func (y *GRPCYavirtd) GetGuestUUID(ctx context.Context, opts *pb.GetGuestOptions) (*pb.GetGuestUUIDMessage, error) {
	log.WithFunc("GRPCYavirtd.GetGuestUUID").Infof(ctx, "get guest UUID: %s", opts.Id)
	guestReq := types.GuestReq{ID: opts.Id}

	uuid, err := y.service.GetGuestUUID(ctx, guestReq.VirtID())
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return &pb.GetGuestUUIDMessage{Uuid: uuid}, nil
}

// CreateGuest .
func (y *GRPCYavirtd) CreateGuest(ctx context.Context, opts *pb.CreateGuestOptions) (*pb.CreateGuestMessage, error) {
	log.WithFunc("GRPCYavirtd.CreateGuest").Infof(ctx, "create guest: %q", opts)
	guest, err := y.service.CreateGuest(ctx, intertypes.ConvertGRPCCreateOptions(opts))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &pb.CreateGuestMessage{
		Id:            types.EruID(guest.ID),
		Status:        guest.Status,
		TransitStatus: guest.TransitStatus,
		CreateTime:    guest.CreateTime,
		TransitTime:   guest.TransitTime,
		UpdateTime:    guest.UpdateTime,
		Cpu:           int64(guest.CPU),
		Memory:        guest.Mem,
		Storage:       guest.Storage,
		ImageId:       guest.ImageID,
		ImageName:     guest.ImageName,
		ImageUser:     guest.ImageUser,
		Networks:      guest.Networks,
	}, nil
}

// CaptureGuest .
func (y *GRPCYavirtd) CaptureGuest(ctx context.Context, opts *pb.CaptureGuestOptions) (*pb.UserImageMessage, error) {
	logger := log.WithFunc("GRPCYavirtd.CaptureGuest").WithField("id", opts.Id)
	logger.Infof(ctx, "capture guest: %q", opts)

	imgName := vmiFact.NewImageName(opts.User, opts.Name)
	uimg, err := y.service.CaptureGuest(ctx, utils.VirtID(opts.Id), imgName, opts.Overridden)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return &pb.UserImageMessage{
		Name:   uimg.Fullname(),
		Distro: uimg.OS.Distrib,
		Size:   uimg.VirtualSize,
	}, nil
}

// ResizeGuest .
func (y *GRPCYavirtd) ResizeGuest(ctx context.Context, opts *pb.ResizeGuestOptions) (*pb.ControlGuestMessage, error) {
	logger := log.WithFunc("GRPCYavirtd.ResizeGuest").WithField("id", opts.Id)
	logger.Infof(ctx, "[grpcserver] resize guest: %q", opts)

	msg := &pb.ControlGuestMessage{Msg: "ok"}

	req := intertypes.ConvertGRPCResizeOptions(opts)
	err := y.service.ResizeGuest(ctx, utils.VirtID(opts.Id), req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// ControlGuest .
func (y *GRPCYavirtd) ControlGuest(ctx context.Context, opts *pb.ControlGuestOptions) (_ *pb.ControlGuestMessage, err error) {
	log.Infof(ctx, "[grpcserver] control guest: %q", opts)
	err = y.service.ControlGuest(ctx, utils.VirtID(opts.Id), opts.Operation, opts.Force)

	msg := "ok"
	if err != nil {
		msg = fmt.Sprintf("%s", err)
	}
	return &pb.ControlGuestMessage{Msg: msg}, errors.Wrap(err, "")
}

// AttachGuest .
func (y *GRPCYavirtd) AttachGuest(server pb.YavirtdRPC_AttachGuestServer) (err error) {
	ctx := server.Context()
	defer log.Info(ctx, "[grpcserver] attach guest complete")
	opts, err := server.Recv()
	if err != nil {
		return
	}
	log.Infof(ctx, "[grpcserver] attach guest start: %v", opts)

	serverStream := &ExecuteGuestServerStream{
		ID:     opts.Id,
		server: server,
	}
	flags := intertypes.NewOpenConsoleFlags(opts.Force, opts.Safe, opts.Commands)
	return y.service.AttachGuest(ctx, utils.VirtID(opts.Id), serverStream, flags)
}

// ResizeConsoleWindow .
func (y *GRPCYavirtd) ResizeConsoleWindow(ctx context.Context, opts *pb.ResizeWindowOptions) (*pb.Empty, error) {
	req := types.GuestReq{ID: opts.Id}
	return nil, y.service.ResizeConsoleWindow(ctx, req.VirtID(), uint(opts.Height), uint(opts.Width))
}

// ExecuteGuest .
func (y *GRPCYavirtd) ExecuteGuest(ctx context.Context, opts *pb.ExecuteGuestOptions) (msg *pb.ExecuteGuestMessage, err error) {
	logger := log.WithFunc("GRPCYavirtd.ExecuteGuest").WithField("id", opts.Id)
	logger.Infof(ctx, "[grpcserver] execute guest start, commands: %s", opts.Commands)
	defer logger.Infof(ctx, "[grpcserver] execute guest done")

	req := types.GuestReq{ID: opts.Id}
	m, err := y.service.ExecuteGuest(ctx, req.VirtID(), opts.Commands)
	if err != nil {
		return
	}
	return &pb.ExecuteGuestMessage{
		Pid:      int64(m.Pid),
		Data:     m.Data,
		ExitCode: int64(m.ExitCode),
	}, nil
}

func (y *GRPCYavirtd) ExecExitCode(ctx context.Context, opts *pb.ExecExitCodeOptions) (msg *pb.ExecExitCodeMessage, err error) {
	log.Infof(ctx, "[grpcserver] get exit code start %q", opts)
	defer log.Infof(ctx, "[grpcserver] get exit code done")

	req := types.GuestReq{ID: opts.Id}

	m, err := y.service.ExecExitCode(req.VirtID(), int(opts.Pid))
	if err != nil {
		return
	}
	return &pb.ExecExitCodeMessage{ExitCode: int64(m)}, nil
}

// ConnectNetwork .
func (y *GRPCYavirtd) ConnectNetwork(ctx context.Context, opts *pb.ConnectNetworkOptions) (*pb.ConnectNetworkMessage, error) {
	log.Infof(ctx, "[grpcserver] connect network start %q", opts)

	req := types.ConnectNetworkReq{
		Network: opts.Network,
		IPv4:    opts.Ipv4,
	}
	req.ID = opts.Id

	cidr, err := y.service.ConnectNetwork(ctx, req.VirtID(), req.Network, req.IPv4)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return &pb.ConnectNetworkMessage{Cidr: cidr}, nil
}

// DisconnectNetwork .
func (y *GRPCYavirtd) DisconnectNetwork(ctx context.Context, opts *pb.DisconnectNetworkOptions) (*pb.DisconnectNetworkMessage, error) {
	log.Infof(ctx, "[grpcserver] disconnect network start")

	var req types.DisconnectNetworkReq
	req.ID = opts.Id
	req.Network = opts.Network

	if err := y.service.DisconnectNetwork(ctx, req.VirtID(), req.Network); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return &pb.DisconnectNetworkMessage{Msg: "ok"}, nil
}

// NetworkList .
func (y *GRPCYavirtd) NetworkList(ctx context.Context, opts *pb.NetworkListOptions) (*pb.NetworkListMessage, error) {
	log.Infof(ctx, "[grpcserver] list network start")
	defer log.Infof(ctx, "[grpcserver] list network completed %v", opts)

	networks, err := y.service.NetworkList(ctx, opts.Drivers)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	msg := &pb.NetworkListMessage{Networks: make(map[string][]byte)}
	for _, network := range networks {
		content, err := json.Marshal(network.Subnets)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		msg.Networks[network.Name] = content
	}

	return msg, nil
}

// Cat .
func (y *GRPCYavirtd) Cat(opts *pb.CatOptions, srv pb.YavirtdRPC_CatServer) error {
	ctx := srv.Context()
	logger := log.WithFunc("GRPCYavirtd.Cat").WithField("id", opts.Id)
	logger.Infof(ctx, "cat %v", opts)
	defer logger.Infof(ctx, "cat %v done", opts)

	req := types.GuestReq{ID: opts.Id}
	wc := &CatWriteCloser{srv: srv}

	err := y.service.Cat(ctx, req.VirtID(), opts.Path, wc)

	return err
}

// CopyToGuest .
func (y *GRPCYavirtd) CopyToGuest(server pb.YavirtdRPC_CopyToGuestServer) (err error) {
	ctx := server.Context()
	logger := log.WithFunc("GRPCYavirtd.CopyToGuest")

	var opts *pb.CopyOptions
	byteChan := make(chan []byte, 4*types.BufferSize)

	opts, err = server.Recv()
	if opts == nil {
		logger.Errorf(ctx, err, "failed to receive options")
		if err != io.EOF {
			return err
		}
		return nil
	}
	logger = logger.WithField("id", opts.Id)
	logger.Info(ctx, "copy file to guest start")
	defer log.Info(ctx, "copy file to guest done")

	req := types.GuestReq{ID: opts.Id}
	dest := opts.Dest
	override := opts.Override
	byteChan <- opts.Content[:opts.Size]

	end := make(chan bool)
	go func() {
		for {
			if opts, err = server.Recv(); err != nil || opts == nil {
				if err == nil {
					err = errors.New("receive no content")
				}
				close(byteChan)
				end <- false
				return
			}

			byteChan <- opts.Content[:opts.Size]
		}
	}()

	if err := y.service.CopyToGuest(ctx, req.VirtID(), dest, byteChan, override); err != nil {
		<-end
		return server.SendAndClose(&pb.CopyMessage{Msg: "copy failed: " + err.Error(), Failed: true})
	}

	if err != nil && err != io.EOF {
		return server.SendAndClose(&pb.CopyMessage{Msg: "copy failed: " + err.Error(), Failed: true})
	}
	return server.SendAndClose(&pb.CopyMessage{Msg: "copy completed", Failed: false})
}

// Log .
func (y *GRPCYavirtd) Log(opts *pb.LogOptions, srv pb.YavirtdRPC_LogServer) error {
	ctx := srv.Context()
	log.Infof(ctx, "[grpcserver] log start")
	defer log.Infof(ctx, "[grpcserver] log completed")

	req := types.GuestReq{ID: opts.Id}
	wc := &LogWriteCloser{srv: srv}
	defer wc.Close()

	return y.service.Log(ctx, req.VirtID(), "/var/log/syslog", int(opts.N), wc)
}

// WaitGuest .
func (y *GRPCYavirtd) WaitGuest(ctx context.Context, opts *pb.WaitGuestOptions) (*pb.WaitGuestMessage, error) {
	log.Infof(ctx, "[grpcserver] wait guest")
	defer log.Infof(ctx, "[grpcserver] wait complete")

	req := types.GuestReq{ID: opts.Id}
	msg, code, err := y.service.Wait(ctx, req.VirtID(), true)
	if err != nil {
		return &pb.WaitGuestMessage{
			Msg:  errors.Wrap(err, "").Error(),
			Code: -1,
		}, errors.Wrap(err, "")
	}

	return &pb.WaitGuestMessage{Msg: msg, Code: int64(code)}, nil
}

// PushImage .
func (y *GRPCYavirtd) PushImage(ctx context.Context, opts *pb.PushImageOptions) (*pb.PushImageMessage, error) {
	log.Infof(ctx, "[grpcserver] PushImage %v", opts)
	defer log.Infof(ctx, "[grpcserver] Push %v completed", opts)

	msg := &pb.PushImageMessage{}

	// TODO add force to opts
	force := false
	imgName := vmiFact.NewImageName(opts.User, opts.ImgName)
	rc, err := y.service.PushImage(ctx, imgName, force)
	if err != nil {
		msg.Err = err.Error()
		return msg, err
	}
	defer utils.EnsureReaderClosed(rc)

	return msg, nil
}

// RemoveImage .
func (y *GRPCYavirtd) RemoveImage(ctx context.Context, opts *pb.RemoveImageOptions) (*pb.RemoveImageMessage, error) {
	log.Infof(ctx, "[grpcserver] RemoveImage %v", opts)
	defer log.Infof(ctx, "[grpcserver] Remove %v completed", opts)

	msg := &pb.RemoveImageMessage{}
	var err error

	imgName := vmiFact.NewImageName(opts.User, opts.Image)
	msg.Removed, err = y.service.RemoveImage(ctx, imgName, opts.Force, opts.Prune)

	return msg, err
}

// ListImage .
func (y *GRPCYavirtd) ListImage(ctx context.Context, opts *pb.ListImageOptions) (*pb.ListImageMessage, error) {
	log.Infof(ctx, "[grpcserver] ListImage %v", opts)
	defer log.Infof(ctx, "[grpcserver] ListImage %v completed", opts)

	imgs, err := y.service.ListImage(ctx, opts.Filter)
	if err != nil {
		return nil, err
	}
	// TODO: remove User in pb
	msg := &pb.ListImageMessage{Images: []*pb.ImageItem{}}
	for _, img := range imgs {
		msg.Images = append(msg.Images, &pb.ImageItem{
			Name:   img.Fullname(),
			Distro: img.OS.Distrib,
		})
	}

	return msg, nil
}

func (y *GRPCYavirtd) PullImage(ctx context.Context, opts *pb.PullImageOptions) (*pb.PullImageMessage, error) {
	log.Infof(ctx, "[grpcserver] PullImage %v", opts)
	defer log.Infof(ctx, "[grpcserver] PullImage %v completed", opts)

	img, rc, err := y.service.PullImage(ctx, opts.Name)
	if err != nil {
		return nil, err
	}
	defer utils.EnsureReaderClosed(rc)

	// TODO change pb to return image
	msg, err := json.Marshal(img)
	if err != nil {
		return nil, err
	}
	return &pb.PullImageMessage{Result: string(msg)}, nil
}

// DigestImage .
func (y *GRPCYavirtd) DigestImage(ctx context.Context, opts *pb.DigestImageOptions) (*pb.DigestImageMessage, error) {
	log.Infof(ctx, "[grpcserver] DigestImage %v", opts)
	defer log.Infof(ctx, "[grpcserver] DigestImage %v completed", opts)

	digests, err := y.service.DigestImage(ctx, opts.ImageName, opts.Local)
	if err != nil {
		return nil, err
	}

	return &pb.DigestImageMessage{Digests: digests}, nil
}

// ListSnapshot .
func (y *GRPCYavirtd) ListSnapshot(ctx context.Context, opts *pb.ListSnapshotOptions) (*pb.ListSnapshotMessage, error) {
	log.Infof(ctx, "[grpcserver] list snapshot: %q", opts)

	req := types.ListSnapshotReq{
		ID:    opts.Id,
		VolID: opts.VolId,
	}

	snaps, err := y.service.ListSnapshot(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	snapshots := []*pb.ListSnapshotMessageItem{}
	for _, snap := range snaps {
		snapshots = append(snapshots, &pb.ListSnapshotMessageItem{
			VolId:       snap.VolID,
			VolMountDir: snap.VolMountDir,
			SnapId:      snap.SnapID,
			CreatedTime: snap.CreatedTime,
		})
	}

	return &pb.ListSnapshotMessage{
		Snapshots: snapshots,
	}, nil
}

// CreateSnapshot .
func (y *GRPCYavirtd) CreateSnapshot(ctx context.Context, opts *pb.CreateSnapshotOptions) (*pb.CreateSnapshotMessage, error) {
	log.Infof(ctx, "[grpcserver] create snapshot: %q", opts)

	msg := &pb.CreateSnapshotMessage{Msg: "ok"}

	req := types.CreateSnapshotReq{
		ID:    opts.Id,
		VolID: opts.VolId,
	}

	err := y.service.CreateSnapshot(ctx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// CommitSnapshot .
func (y *GRPCYavirtd) CommitSnapshot(ctx context.Context, opts *pb.CommitSnapshotOptions) (*pb.CommitSnapshotMessage, error) {
	log.Infof(ctx, "[grpcserver] commit snapshot: %q", opts)

	msg := &pb.CommitSnapshotMessage{Msg: "ok"}

	req := types.CommitSnapshotReq{
		ID:     opts.Id,
		VolID:  opts.VolId,
		SnapID: opts.SnapId,
	}

	err := y.service.CommitSnapshot(ctx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// RestoreSnapshot .
func (y *GRPCYavirtd) RestoreSnapshot(ctx context.Context, opts *pb.RestoreSnapshotOptions) (*pb.RestoreSnapshotMessage, error) {
	log.Infof(ctx, "[grpcserver] restore snapshot: %q", opts)

	msg := &pb.RestoreSnapshotMessage{Msg: "ok"}

	req := types.RestoreSnapshotReq{
		ID:     opts.Id,
		VolID:  opts.VolId,
		SnapID: opts.SnapId,
	}

	err := y.service.RestoreSnapshot(ctx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// ExecuteGuest .
func (y *GRPCYavirtd) RawEngine(ctx context.Context, opts *pb.RawEngineOptions) (msg *pb.RawEngineMessage, err error) {
	logger := log.WithFunc("RawEngine").WithField("id", opts.Id).WithField("op", opts.Op)
	logger.Infof(ctx, "[grpcserver] raw engine operation, params: %s", string(opts.Params))
	req := types.RawEngineReq{
		ID:     opts.Id,
		Op:     opts.Op,
		Params: opts.Params,
	}
	m, err := y.service.RawEngine(ctx, utils.VirtID(opts.Id), req)
	if err != nil {
		return
	}
	return &pb.RawEngineMessage{
		Id:   opts.Id,
		Data: m.Data,
	}, nil
}
