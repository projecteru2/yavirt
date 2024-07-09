package grpcserver

import (
	"context"
	"crypto/tls"
	"net"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/projecteru2/core/auth"
	"github.com/projecteru2/core/log"
	pb "github.com/projecteru2/libyavirt/grpc/gen"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// GRPCServer .
type GRPCServer struct {
	server *grpc.Server
	app    pb.YavirtdRPCServer
}

func loadTLSCredentials(dir string) (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	certFile := filepath.Join(dir, "server-cert.pem")
	keyFile := filepath.Join(dir, "server-key.pem")
	if (!utils.FileExists(certFile)) || (!utils.FileExists(keyFile)) {
		return nil, nil //nolint
	}
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	return credentials.NewTLS(config), nil
}

func New(cfg *configs.Config, svc service.Service) (*GRPCServer, error) {
	logger := log.WithFunc("rpc.New")
	opts := []grpc.ServerOption{}
	certDir := filepath.Join(cfg.CertPath, "yavirt")
	tlsCredentials, err := loadTLSCredentials(certDir)
	if err != nil {
		return nil, err
	}
	if tlsCredentials != nil {
		logger.Info(context.TODO(), "grpc server tls enabled")
		opts = append(opts, grpc.Creds(tlsCredentials))
	}
	if cfg.Auth.Username != "" {
		logger.Info(context.TODO(), "grpc server auth enabled")
		auth := auth.NewAuth(cfg.Auth)
		opts = append(opts, grpc.StreamInterceptor(auth.StreamInterceptor))
		opts = append(opts, grpc.UnaryInterceptor(auth.UnaryInterceptor))
		logger.Debugf(context.TODO(), "username %s password %s", cfg.Auth.Username, cfg.Auth.Password)
	}
	srv := &GRPCServer{
		server: grpc.NewServer(opts...),
		app:    &GRPCYavirtd{service: svc},
	}
	reflection.Register(srv.server)

	return srv, nil
}

// Serve .
func (s *GRPCServer) Serve() error {
	defer func() {
		log.WithFunc("rpc.Serve").Warnf(context.TODO(), "[grpcserver] main loop %p exit", s)
	}()
	lis, err := net.Listen("tcp", configs.Conf.BindGRPCAddr)
	if err != nil {
		return err
	}
	pb.RegisterYavirtdRPCServer(s.server, s.app)

	return s.server.Serve(lis)
}

// Close .
func (s *GRPCServer) Stop(force bool) {
	logger := log.WithFunc("rpc.Close")
	if force {
		logger.Warnf(context.TODO(), "[grpcserver] terminate grpc server forcefully")
		s.server.Stop()
		return
	}

	gracefulDone := make(chan struct{})
	go func() {
		defer close(gracefulDone)
		s.server.GracefulStop()
	}()

	gracefulTimer := time.NewTimer(configs.Conf.GracefulTimeout)
	select {
	case <-gracefulDone:
		logger.Infof(context.TODO(), "[grpcserver] terminate grpc server gracefully")
	case <-gracefulTimer.C:
		logger.Warnf(context.TODO(), "[grpcserver] terminate grpc server forcefully")
		s.server.Stop()
	}
}
