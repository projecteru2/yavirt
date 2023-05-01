package httpserver

import (
	"context"
	"net/http"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/server"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// HTTPServer .
type HTTPServer struct {
	*server.Server

	httpServer *http.Server
}

// Listen .
func Listen(svc *server.Service) (srv *HTTPServer, err error) {
	srv = &HTTPServer{}
	if srv.Server, err = server.Listen(configs.Conf.BindHTTPAddr, svc); err != nil {
		return
	}

	srv.httpServer = srv.newHTTPServer()

	return srv, nil
}

func (s *HTTPServer) newHTTPServer() *http.Server {
	var mux = http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", newAPIHandler(s.Service))
	return &http.Server{Handler: mux} //nolint
}

// Reload .
func (s *HTTPServer) Reload() error {
	return nil
}

// Serve .
func (s *HTTPServer) Serve() (err error) {
	defer func() {
		log.Warnf("[httpserver] main loop %p exit", s)
		s.Close()
	}()

	var errCh = make(chan error, 1)
	go func() {
		defer func() {
			log.Warnf("[httpserver] HTTP server %p exit", s.httpServer)
		}()
		errCh <- s.httpServer.Serve(s.Listener)
	}()

	select {
	case <-s.Exit.Ch:
		return nil
	case err = <-errCh:
		return errors.Trace(err)
	}
}

// Close .
func (s *HTTPServer) Close() {
	s.Exit.Do(func() {
		close(s.Exit.Ch)

		var err error
		defer func() {
			if err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}()

		var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GracefulTimeout.Duration())
		defer cancel()

		if err = s.httpServer.Shutdown(ctx); err != nil {
			return
		}
	})
}

// ExitCh .
func (s *HTTPServer) ExitCh() chan struct{} {
	return s.Exit.Ch
}
