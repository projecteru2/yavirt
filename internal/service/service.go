package service

import (
	"context"
	"io"

	"github.com/projecteru2/libyavirt/types"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/utils"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

// Service interface
// Note: all ids passed to this interface and returned by this interface don't contains maigc prefix
type Service interface { //nolint:interfacebloat
	Ping() map[string]string
	Info() (*types.HostInfo, error)
	IsHealthy(ctx context.Context) (ans bool)

	// Guest related functions
	GetGuest(ctx context.Context, id string) (*types.Guest, error)
	GetGuestIDList(ctx context.Context) ([]string, error)
	GetGuestUUID(ctx context.Context, id string) (string, error)
	CreateGuest(ctx context.Context, opts intertypes.GuestCreateOption) (*types.Guest, error)
	CaptureGuest(ctx context.Context, id string, imgName string, overridden bool) (uimg *vmitypes.Image, err error)
	ResizeGuest(ctx context.Context, id string, opts *intertypes.GuestResizeOption) (err error)
	ControlGuest(ctx context.Context, id, operation string, force bool) (err error)
	AttachGuest(ctx context.Context, id string, stream io.ReadWriteCloser, flags intertypes.OpenConsoleFlags) (err error)
	ResizeConsoleWindow(ctx context.Context, id string, height, width uint) (err error)
	Wait(ctx context.Context, id string, block bool) (msg string, code int, err error)
	WatchGuestEvents(context.Context) (*utils.Watcher, error)

	// Guest utilities
	ExecuteGuest(ctx context.Context, id string, commands []string) (*types.ExecuteGuestMessage, error)
	ExecExitCode(id string, pid int) (int, error)
	Cat(ctx context.Context, id, path string, dest io.WriteCloser) (err error)
	CopyToGuest(ctx context.Context, id, dest string, content chan []byte, override bool) (err error)
	Log(ctx context.Context, id, logPath string, n int, dest io.WriteCloser) (err error)

	// Snapshot
	ListSnapshot(ctx context.Context, req types.ListSnapshotReq) (snaps types.Snapshots, err error)
	CreateSnapshot(ctx context.Context, req types.CreateSnapshotReq) (err error)
	CommitSnapshot(ctx context.Context, req types.CommitSnapshotReq) (err error)
	CommitSnapshotByDay(ctx context.Context, id, volID string, day int) (err error)
	RestoreSnapshot(ctx context.Context, req types.RestoreSnapshotReq) (err error)

	// Network
	NetworkList(ctx context.Context, drivers []string) ([]*types.Network, error)
	ConnectNetwork(ctx context.Context, id, network, ipv4 string) (cidr string, err error)
	DisconnectNetwork(ctx context.Context, id, network string) (err error)

	// Image
	PushImage(ctx context.Context, imgName string, force bool) (rc io.ReadCloser, err error)
	RemoveImage(ctx context.Context, imageName string, force, prune bool) (removed []string, err error)
	ListImage(ctx context.Context, filter string) ([]*vmitypes.Image, error)
	PullImage(ctx context.Context, imgName string) (img *vmitypes.Image, rc io.ReadCloser, err error)
	DigestImage(ctx context.Context, imageName string, local bool) (digest []string, err error)

	RawEngine(ctx context.Context, id string, req types.RawEngineReq) (types.RawEngineResp, error)
}
