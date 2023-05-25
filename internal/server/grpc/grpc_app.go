package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	pb "github.com/projecteru2/libyavirt/grpc/gen"
	"github.com/projecteru2/libyavirt/types"

	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"

	"github.com/projecteru2/yavirt/internal/server"
)

// GRPCYavirtd .
type GRPCYavirtd struct {
	service *server.Service
}

// Ping .
func (y *GRPCYavirtd) Ping(_ context.Context, _ *pb.Empty) (*pb.PingMessage, error) {
	pang := y.service.Ping()
	return &pb.PingMessage{Version: pang["version"]}, nil
}

// GetInfo .
func (y *GRPCYavirtd) GetInfo(_ context.Context, _ *pb.Empty) (*pb.InfoMessage, error) {
	log.Infof("[grpcserver] get host info")
	info := y.service.Info()
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
	log.Infof("[grpcserver] get guest: %s", opts.Id)
	guestReq := types.GuestReq{ID: opts.Id}
	guest, err := y.service.GetGuest(y.service.VirtContext(ctx), guestReq.VirtID())
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &pb.GetGuestMessage{
		Id:            guest.ID,
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
	}, nil
}

// GetGuestIDList gets all local vms' domain names regardless of their metadata validility.
func (y *GRPCYavirtd) GetGuestIDList(ctx context.Context, _ *pb.GetGuestIDListOptions) (*pb.GetGuestIDListMessage, error) {
	log.Infof("[grpcserver] get guest id list")
	ids, err := y.service.GetGuestIDList(y.service.VirtContext(ctx))
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &pb.GetGuestIDListMessage{Ids: ids}, nil
}

// Events
func (y *GRPCYavirtd) Events(_ *pb.EventsOptions, server pb.YavirtdRPC_EventsServer) error {
	log.Infof("[grpcserver] events method calling")
	defer log.Infof("[grpcserver] events method completed")

	ctx := server.Context()
	watcher, err := y.service.WatchGuestEvents(y.service.VirtContext(ctx))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Infof("[grpcserver] events goroutine has done")
		defer watcher.Stop()

		for {
			select {
			case event := <-watcher.Events():
				if err := server.Send(parseEvent(event)); err != nil {
					log.ErrorStack(err)
					return
				}

			case <-watcher.Done():
				// The watcher already has been stopped.
				log.Infof("[grpcserver] watcher has done")
				return

			case <-ctx.Done():
				log.Infof("[grpcserver] ctx done")
				return
			}
		}
	}()

	return nil
}

func parseEvent(event virtypes.Event) *pb.EventMessage {
	return &pb.EventMessage{
		Id:       types.EruID(event.ID),
		Type:     event.Type,
		Action:   event.Action,
		TimeNano: event.Time.UnixNano(),
	}
}

// GetGuestUUID .
func (y *GRPCYavirtd) GetGuestUUID(ctx context.Context, opts *pb.GetGuestOptions) (*pb.GetGuestUUIDMessage, error) {
	log.Infof("[grpcserver] get guest UUID: %s", opts.Id)
	guestReq := types.GuestReq{ID: opts.Id}

	uuid, err := y.service.GetGuestUUID(y.service.VirtContext(ctx), guestReq.VirtID())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.GetGuestUUIDMessage{Uuid: uuid}, nil
}

// CreateGuest .
func (y *GRPCYavirtd) CreateGuest(ctx context.Context, opts *pb.CreateGuestOptions) (*pb.CreateGuestMessage, error) {
	log.Infof("[grpcserver] create guest: %q", opts)
	guest, err := y.service.CreateGuest(y.service.VirtContext(ctx), virtypes.ConvertGRPCCreateOptions(opts))
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &pb.CreateGuestMessage{
		Id:            guest.ID,
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
	log.Infof("[grpcserver] capture guest: %q", opts)

	req := types.CaptureGuestReq{
		Name:       opts.Name,
		User:       opts.User,
		Overridden: opts.Overridden,
	}
	req.ID = opts.Id

	virtCtx := y.service.VirtContext(ctx)

	uimg, err := y.service.CaptureGuest(virtCtx, req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.UserImageMessage{
		Id:            uimg.GetID(),
		Name:          uimg.Name,
		Distro:        uimg.Distro,
		LatestVersion: uimg.Version,
		Size:          uimg.Size,
	}, nil
}

// ResizeGuest .
func (y *GRPCYavirtd) ResizeGuest(ctx context.Context, opts *pb.ResizeGuestOptions) (*pb.ControlGuestMessage, error) {
	log.Infof("[grpcserver] resize guest: %q", opts)

	msg := &pb.ControlGuestMessage{Msg: "ok"}
	virtCtx := y.service.VirtContext(ctx)

	req := types.ResizeGuestReq{
		CPU:       int(opts.Cpu),
		Mem:       opts.Memory,
		Resources: opts.Resources,
	}
	req.Volumes = make([]types.Volume, len(opts.Volumes))
	for i, vol := range opts.Volumes {
		req.Volumes[i].Mount = vol.Mount
		req.Volumes[i].Capacity = vol.Capacity
		req.Volumes[i].IO = vol.Io
	}
	req.ID = opts.Id

	err := y.service.ResizeGuest(virtCtx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// ControlGuest .
func (y *GRPCYavirtd) ControlGuest(ctx context.Context, opts *pb.ControlGuestOptions) (_ *pb.ControlGuestMessage, err error) {
	log.Infof("[grpcserver] control guest: %q", opts)
	req := types.GuestReq{ID: opts.Id}
	virtCtx := y.service.VirtContext(ctx)
	err = y.service.ControlGuest(virtCtx, req.VirtID(), opts.Operation, opts.Force)

	msg := "ok"
	if err != nil {
		msg = fmt.Sprintf("%s", err)
	}
	return &pb.ControlGuestMessage{Msg: msg}, errors.Trace(err)
}

// AttachGuest .
func (y *GRPCYavirtd) AttachGuest(server pb.YavirtdRPC_AttachGuestServer) (err error) {
	defer log.Infof("[grpcserver] attach guest complete")
	log.Infof("[grpcserver] attach guest start")
	opts, err := server.Recv()
	if err != nil {
		return
	}

	virtCtx := y.service.VirtContext(server.Context())
	req := types.GuestReq{ID: opts.Id}
	serverStream := &ExecuteGuestServerStream{
		ID:     opts.Id,
		server: server,
	}
	flags := virtypes.OpenConsoleFlags{Force: opts.Force, Safe: opts.Safe, Commands: opts.Commands}
	return y.service.AttachGuest(virtCtx, req.VirtID(), serverStream, flags)
}

// ResizeConsoleWindow .
func (y *GRPCYavirtd) ResizeConsoleWindow(ctx context.Context, opts *pb.ResizeWindowOptions) (*pb.Empty, error) {
	req := types.GuestReq{ID: opts.Id}
	virtCtx := y.service.VirtContext(ctx)
	return nil, y.service.ResizeConsoleWindow(virtCtx, req.VirtID(), uint(opts.Height), uint(opts.Width))
}

// ExecuteGuest .
func (y *GRPCYavirtd) ExecuteGuest(ctx context.Context, opts *pb.ExecuteGuestOptions) (msg *pb.ExecuteGuestMessage, err error) {
	log.Infof("[grpcserver] execute guest start")
	req := types.GuestReq{ID: opts.Id}
	virtCtx := y.service.VirtContext(ctx)
	m, err := y.service.ExecuteGuest(virtCtx, req.VirtID(), opts.Commands)
	if err != nil {
		return
	}
	return &pb.ExecuteGuestMessage{
		Pid:      int64(m.Pid),
		Data:     m.Data,
		ExitCode: int64(m.ExitCode),
	}, nil
}

func (y *GRPCYavirtd) ExecExitCode(_ context.Context, opts *pb.ExecExitCodeOptions) (msg *pb.ExecExitCodeMessage, err error) {
	log.Infof("[grpcserver] get exit code start")
	defer log.Infof("[grpcserver] get exit code done")

	req := types.GuestReq{ID: opts.Id}

	m, err := y.service.ExecExitCode(req.VirtID(), int(opts.Pid))
	if err != nil {
		return
	}
	return &pb.ExecExitCodeMessage{ExitCode: int64(m)}, nil
}

// ConnectNetwork .
func (y *GRPCYavirtd) ConnectNetwork(ctx context.Context, opts *pb.ConnectNetworkOptions) (*pb.ConnectNetworkMessage, error) {
	log.Infof("[grpcserver] connect network start")

	req := types.ConnectNetworkReq{
		Network: opts.Network,
		IPv4:    opts.Ipv4,
	}
	req.ID = opts.Id

	virtCtx := y.service.VirtContext(ctx)
	cidr, err := y.service.ConnectNetwork(virtCtx, req.VirtID(), req.Network, req.IPv4)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.ConnectNetworkMessage{Cidr: cidr}, nil
}

// DisconnectNetwork .
func (y *GRPCYavirtd) DisconnectNetwork(ctx context.Context, opts *pb.DisconnectNetworkOptions) (*pb.DisconnectNetworkMessage, error) {
	log.Infof("[grpcserver] disconnect network start")

	var req types.DisconnectNetworkReq
	req.ID = opts.Id
	req.Network = opts.Network

	virtCtx := y.service.VirtContext(ctx)
	if err := y.service.DisconnectNetwork(virtCtx, req.VirtID(), req.Network); err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.DisconnectNetworkMessage{Msg: "ok"}, nil
}

// NetworkList .
func (y *GRPCYavirtd) NetworkList(ctx context.Context, opts *pb.NetworkListOptions) (*pb.NetworkListMessage, error) {
	log.Infof("[grpcserver] list network start")
	defer log.Infof("[grpcserver] list network completed %v", opts)

	virtCtx := y.service.VirtContext(ctx)
	networks, err := y.service.NetworkList(virtCtx, opts.Drivers)
	if err != nil {
		return nil, errors.Trace(err)
	}

	msg := &pb.NetworkListMessage{Networks: make(map[string][]byte)}
	for _, network := range networks {
		content, err := json.Marshal(network.Subnets)
		if err != nil {
			return nil, errors.Trace(err)
		}
		msg.Networks[network.Name] = content
	}

	return msg, nil
}

// Cat .
func (y *GRPCYavirtd) Cat(opts *pb.CatOptions, srv pb.YavirtdRPC_CatServer) error {
	log.Infof("[grpcserver] cat %v", opts)
	defer log.Infof("[grpcserver] cat %v completed", opts)

	ctx := y.service.VirtContext(srv.Context())
	req := types.GuestReq{ID: opts.Id}
	wc := &CatWriteCloser{srv: srv}

	err := y.service.Cat(ctx, req.VirtID(), opts.Path, wc)

	return err
}

// CopyToGuest .
func (y *GRPCYavirtd) CopyToGuest(server pb.YavirtdRPC_CopyToGuestServer) (err error) {
	defer log.Infof("[grpcserver] copy file to guest complete")
	log.Infof("[grpcserver] copy file to guest start")

	var opts *pb.CopyOptions
	byteChan := make(chan []byte, 4*types.BufferSize)

	opts, err = server.Recv()
	if opts == nil {
		if err != io.EOF {
			return err
		}
		return nil
	}
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

	ctx := y.service.VirtContext(server.Context())
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
	log.Infof("[grpcserver] log start")
	defer log.Infof("[grpcserver] log completed")

	virtCtx := y.service.VirtContext(srv.Context())
	req := types.GuestReq{ID: opts.Id}
	wc := &LogWriteCloser{srv: srv}
	defer wc.Close()

	return y.service.Log(virtCtx, req.VirtID(), "/var/log/syslog", int(opts.N), wc)
}

// WaitGuest .
func (y *GRPCYavirtd) WaitGuest(ctx context.Context, opts *pb.WaitGuestOptions) (*pb.WaitGuestMessage, error) {
	log.Infof("[grpcserver] wait guest")
	defer log.Infof("[grpcserver] wait complete")

	req := types.GuestReq{ID: opts.Id}
	virtCtx := y.service.VirtContext(ctx)
	msg, code, err := y.service.Wait(virtCtx, req.VirtID(), true)
	if err != nil {
		return &pb.WaitGuestMessage{
			Msg:  errors.Trace(err).Error(),
			Code: -1,
		}, errors.Trace(err)
	}

	return &pb.WaitGuestMessage{Msg: msg, Code: int64(code)}, nil
}

// PushImage .
func (y *GRPCYavirtd) PushImage(ctx context.Context, opts *pb.PushImageOptions) (*pb.PushImageMessage, error) {
	log.Infof("[grpcserver] PushImage %v", opts)
	defer log.Infof("[grpcserver] Push %v completed", opts)

	virtCtx := y.service.VirtContext(ctx)
	msg := &pb.PushImageMessage{}

	if err := y.service.PushImage(virtCtx, opts.ImgName, opts.User); err != nil {
		msg.Err = err.Error()
		return msg, err
	}

	return msg, nil
}

// RemoveImage .
func (y *GRPCYavirtd) RemoveImage(ctx context.Context, opts *pb.RemoveImageOptions) (*pb.RemoveImageMessage, error) {
	log.Infof("[grpcserver] RemoveImage %v", opts)
	defer log.Infof("[grpcserver] Remove %v completed", opts)

	virtCtx := y.service.VirtContext(ctx)
	msg := &pb.RemoveImageMessage{}
	var err error

	msg.Removed, err = y.service.RemoveImage(virtCtx, opts.Image, opts.User, opts.Force, opts.Prune)

	return msg, err
}

// ListImage .
func (y *GRPCYavirtd) ListImage(ctx context.Context, opts *pb.ListImageOptions) (*pb.ListImageMessage, error) {
	log.Infof("[grpcserver] ListImage %v", opts)
	defer log.Infof("[grpcserver] ListImage %v completed", opts)

	virtCtx := y.service.VirtContext(ctx)

	imgs, err := y.service.ListImage(virtCtx, opts.Filter)
	if err != nil {
		return nil, err
	}

	msg := &pb.ListImageMessage{Images: []*pb.ImageItem{}}
	for _, img := range imgs {
		msg.Images = append(msg.Images, types.ToGRPCImageItem(img))
	}

	return msg, nil
}

func (y *GRPCYavirtd) PullImage(ctx context.Context, opts *pb.PullImageOptions) (*pb.PullImageMessage, error) {
	log.Infof("[grpcserver] PullImage %v", opts)
	defer log.Infof("[grpcserver] PullImage %v completed", opts)

	virtCtx := y.service.VirtContext(ctx)

	msg, err := y.service.PullImage(virtCtx, opts.Name, opts.All)
	if err != nil {
		return nil, err
	}

	return &pb.PullImageMessage{Result: msg}, nil
}

// DigestImage .
func (y *GRPCYavirtd) DigestImage(ctx context.Context, opts *pb.DigestImageOptions) (*pb.DigestImageMessage, error) {
	log.Infof("[grpcserver] DigestImage %v", opts)
	defer log.Infof("[grpcserver] DigestImage %v completed", opts)

	virtCtx := y.service.VirtContext(ctx)

	digests, err := y.service.DigestImage(virtCtx, opts.ImageName, opts.Local)
	if err != nil {
		return nil, err
	}

	return &pb.DigestImageMessage{Digests: digests}, nil
}

// ListSnapshot .
func (y *GRPCYavirtd) ListSnapshot(ctx context.Context, opts *pb.ListSnapshotOptions) (*pb.ListSnapshotMessage, error) {
	log.Infof("[grpcserver] list snapshot: %q", opts)

	virtCtx := y.service.VirtContext(ctx)

	req := types.ListSnapshotReq{
		ID:    opts.Id,
		VolID: opts.VolId,
	}

	snaps, err := y.service.ListSnapshot(virtCtx, req)
	if err != nil {
		return nil, errors.Trace(err)
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
	log.Infof("[grpcserver] create snapshot: %q", opts)

	msg := &pb.CreateSnapshotMessage{Msg: "ok"}
	virtCtx := y.service.VirtContext(ctx)

	req := types.CreateSnapshotReq{
		ID:    opts.Id,
		VolID: opts.VolId,
	}

	err := y.service.CreateSnapshot(virtCtx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// CommitSnapshot .
func (y *GRPCYavirtd) CommitSnapshot(ctx context.Context, opts *pb.CommitSnapshotOptions) (*pb.CommitSnapshotMessage, error) {
	log.Infof("[grpcserver] commit snapshot: %q", opts)

	msg := &pb.CommitSnapshotMessage{Msg: "ok"}
	virtCtx := y.service.VirtContext(ctx)

	req := types.CommitSnapshotReq{
		ID:     opts.Id,
		VolID:  opts.VolId,
		SnapID: opts.SnapId,
	}

	err := y.service.CommitSnapshot(virtCtx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}

// RestoreSnapshot .
func (y *GRPCYavirtd) RestoreSnapshot(ctx context.Context, opts *pb.RestoreSnapshotOptions) (*pb.RestoreSnapshotMessage, error) {
	log.Infof("[grpcserver] restore snapshot: %q", opts)

	msg := &pb.RestoreSnapshotMessage{Msg: "ok"}
	virtCtx := y.service.VirtContext(ctx)

	req := types.RestoreSnapshotReq{
		ID:     opts.Id,
		VolID:  opts.VolId,
		SnapID: opts.SnapId,
	}

	err := y.service.RestoreSnapshot(virtCtx, req)
	if err != nil {
		msg.Msg = fmt.Sprintf("%s", err)
	}

	return msg, err
}
