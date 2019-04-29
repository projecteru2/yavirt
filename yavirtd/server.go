package yavirtd

import (
	"net"
	"sync"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/netx"
)

// Server .
type Server interface {
	Reload() error
	Serve() error
	Close()
	ExitCh() chan struct{}
}

// ServerBase .
type ServerBase struct {
	Addr     string
	Listener net.Listener
	Service  *Service
	Exit     struct {
		sync.Once
		Ch chan struct{}
	}
}

// Listen .
func Listen(addr string, svc *Service) (srv *ServerBase, err error) {
	srv = &ServerBase{Service: svc}
	srv.Exit.Ch = make(chan struct{}, 1)
	srv.Listener, srv.Addr, err = srv.Listen(addr)
	return
}

// Listen .
func (s *ServerBase) Listen(addr string) (lis net.Listener, ip string, err error) {
	var network = "tcp"
	if lis, err = net.Listen(network, addr); err != nil {
		return
	}

	if ip, err = netx.GetOutboundIP(config.Conf.CoreAddr); err != nil {
		return
	}

	return
}
