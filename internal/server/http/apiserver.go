package httpserver

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/server"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func newAPIHandler(svc *server.Service) http.Handler {
	gin.SetMode(gin.ReleaseMode)

	var api = &apiServer{service: svc}
	var router = gin.Default()

	var v1 = router.Group("/v1")
	{
		v1.GET("/ping", api.Ping)
		v1.GET("/info", api.Info)
		v1.GET("/guests/:id", api.GetGuest)
		v1.GET("/guests/:id/uuid", api.GetGuestUUID)
		v1.POST("/guests", api.CreateGuest)
		v1.POST("/guests/stop", api.StopGuest)
		v1.POST("/guests/start", api.StartGuest)
		v1.POST("/guests/destroy", api.DestroyGuest)
		v1.POST("/guests/execute", api.ExecuteGuest)
		v1.POST("/guests/resize", api.ResizeGuest)
		v1.POST("/guests/capture", api.CaptureGuest)
		v1.POST("/guests/connect", api.ConnectNetwork)
		v1.POST("/guests/resize_window", api.ResizeConsoleWindow)
		// v1.POST("/guests/snapshot/list", api.ListSnapshot)
		v1.POST("/guests/snapshot/create", api.CreateSnapshot)
		v1.POST("/guests/snapshot/commit", api.CommitSnapshot)
		v1.POST("/guests/snapshot/restore", api.RestoreSnapshot)
	}

	return router
}

type apiServer struct {
	service *server.Service
}

func (s *apiServer) host() *models.Host { //nolint
	return s.service.Host
}

func (s *apiServer) Info(c *gin.Context) {
	s.renderOK(c, s.service.Info())
}

func (s *apiServer) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, s.service.Ping())
}

func (s *apiServer) dispatchMsg(c *gin.Context, req interface{}, fn func(virt.Context) error) {
	s.dispatch(c, req, func(ctx virt.Context) (interface{}, error) {
		return nil, fn(ctx)
	})
}

type operate func(virt.Context) (interface{}, error)

func (s *apiServer) dispatch(c *gin.Context, req interface{}, fn operate) {
	if err := s.bind(c, req); err != nil {
		s.renderErr(c, err)
		return
	}

	var resp, err = fn(s.virtContext())
	if err != nil {
		s.renderErr(c, err)
		return
	}

	if resp == nil {
		s.renderOKMsg(c)
	} else {
		s.renderOK(c, resp)
	}
}

func (s *apiServer) bind(c *gin.Context, req interface{}) error {
	switch c.Request.Method {
	case http.MethodGet:
		return c.ShouldBindUri(req)

	case http.MethodPost:
		return c.ShouldBind(req)

	default:
		return errors.Errorf("invalid HTTP method: %s", c.Request.Method)
	}
}

var okMsg = types.NewMsg("ok")

func (s *apiServer) renderOKMsg(c *gin.Context) {
	s.renderOK(c, okMsg)
}

func (s *apiServer) renderOK(c *gin.Context, resp interface{}) {
	c.JSON(http.StatusOK, resp)
}

func (s *apiServer) renderErr(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, err.Error())
}

func (s *apiServer) virtContext() virt.Context {
	return s.service.VirtContext(context.Background())
}
