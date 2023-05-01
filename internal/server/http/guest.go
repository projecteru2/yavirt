package httpserver

import (
	"github.com/gin-gonic/gin"

	"github.com/projecteru2/libyavirt/types"

	"github.com/projecteru2/yavirt/internal/virt"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
)

func (s *apiServer) GetGuest(c *gin.Context) {
	var req types.GuestReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.GetGuest(ctx, req.VirtID())
	})
}

func (s *apiServer) GetGuestUUID(c *gin.Context) {
	var req types.GuestReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.GetGuestUUID(ctx, req.VirtID())
	})
}

func (s *apiServer) CaptureGuest(c *gin.Context) {
	var req types.CaptureGuestReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.CaptureGuest(ctx, req)
	})
}

func (s *apiServer) ResizeGuest(c *gin.Context) {
	var req types.ResizeGuestReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.ResizeGuest(ctx, req)
	})
}

func (s *apiServer) DestroyGuest(c *gin.Context) {
	var req types.GuestReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.ControlGuest(ctx, req.VirtID(), types.OpDestroy, req.Force)
	})
}

func (s *apiServer) StopGuest(c *gin.Context) {
	var req types.GuestReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.ControlGuest(ctx, req.VirtID(), types.OpStop, req.Force)
	})
}

func (s *apiServer) StartGuest(c *gin.Context) {
	var req types.GuestReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.ControlGuest(ctx, req.VirtID(), types.OpStart, false)
	})
}

func (s *apiServer) CreateGuest(c *gin.Context) {
	var req types.CreateGuestReq

	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.CreateGuest(
			ctx,
			virtypes.GuestCreateOption{
				CPU:       req.CPU,
				Mem:       req.Mem,
				ImageName: req.ImageName,
				Volumes:   req.Volumes,
				DmiUUID:   req.DmiUUID,
				Labels:    req.Labels,
				ImageUser: req.ImageUser,
			},
		)
	})
}

func (s *apiServer) ExecuteGuest(c *gin.Context) {
	var req types.ExecuteGuestReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.ExecuteGuest(ctx, req.VirtID(), req.Commands)
	})
}

func (s *apiServer) ConnectNetwork(c *gin.Context) {
	var req types.ConnectNetworkReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.ConnectNetwork(ctx, req.VirtID(), req.Network, req.IPv4)
	})
}

func (s *apiServer) ResizeConsoleWindow(c *gin.Context) {
	var req types.ResizeConsoleWindowReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return nil, s.service.ResizeConsoleWindow(ctx, req.VirtID(), req.Height, req.Width)
	})
}

func (s *apiServer) ListSnapshot(c *gin.Context) {
	var req types.ListSnapshotReq
	s.dispatch(c, &req, func(ctx virt.Context) (any, error) {
		return s.service.ListSnapshot(ctx, req)
	})
}

func (s *apiServer) CreateSnapshot(c *gin.Context) {
	var req types.CreateSnapshotReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.CreateSnapshot(ctx, req)
	})
}

func (s *apiServer) CommitSnapshot(c *gin.Context) {
	var req types.CommitSnapshotReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.CommitSnapshot(ctx, req)
	})
}

func (s *apiServer) RestoreSnapshot(c *gin.Context) {
	var req types.RestoreSnapshotReq
	s.dispatchMsg(c, &req, func(ctx virt.Context) error {
		return s.service.RestoreSnapshot(ctx, req)
	})
}
