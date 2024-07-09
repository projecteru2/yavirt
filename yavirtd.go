package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/debug"
	"github.com/projecteru2/yavirt/internal/metrics"
	grpcserver "github.com/projecteru2/yavirt/internal/rpc"
	"github.com/projecteru2/yavirt/internal/service/boar"
	"github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt"
	zerolog "github.com/rs/zerolog/log"
	cli "github.com/urfave/cli/v2"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	utils.EnforceRoot()

	cli.VersionPrinter = func(_ *cli.Context) {
		fmt.Println(ver.Version())
	}

	app := &cli.App{
		Name:    "yavirtd",
		Usage:   "yavirt daemon",
		Version: "v",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Value:   "/etc/eru/yavirtd.toml",
				Usage:   "config file path for yavirt, in toml",
				EnvVars: []string{"ERU_YAVIRT_CONFIG_PATH"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "",
				Usage:   "set log level",
				EnvVars: []string{"ERU_YAVIRT_LOG_LEVEL"},
			},
			&cli.StringSliceFlag{
				Name:    "core-addrs",
				Value:   cli.NewStringSlice(),
				Usage:   "core addresses",
				EnvVars: []string{"ERU_YAVIRT_CORE_ADDRS"},
			},
			&cli.StringFlag{
				Name:    "core-username",
				Value:   "",
				Usage:   "core username",
				EnvVars: []string{"ERU_YAVIRT_CORE_USERNAME"},
			},
			&cli.StringFlag{
				Name:    "core-password",
				Value:   "",
				Usage:   "core password",
				EnvVars: []string{"ERU_YAVIRT_CORE_PASSWORD"},
			},
			&cli.StringFlag{
				Name:    "hostname",
				Value:   "",
				Usage:   "change hostname",
				EnvVars: []string{"ERU_HOSTNAME", "HOSTNAME"},
			},
		},
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(context.TODO(), err, "run failed")
		os.Exit(1)
	}

	os.Exit(0)
}

func initConfig(c *cli.Context) error {
	cfg := &configs.Conf
	if err := cfg.Load(c.String("config")); err != nil {
		return err
	}
	return cfg.Prepare(c)
}

func startHTTPServer(addr string) {
	http.Handle("/metrics", metrics.Handler())
	http.HandleFunc("/debug/custom", debug.Handler)
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Error(context.TODO(), err, "start http failed")
	}
}

// Run .
func Run(c *cli.Context) error {
	if err := initConfig(c); err != nil {
		zerolog.Fatal().Err(err).Msg("invalid config")
		return err
	}
	svcLog := &coretypes.ServerLogConfig{
		Level:      configs.Conf.Log.Level,
		UseJSON:    configs.Conf.Log.UseJSON,
		Filename:   configs.Conf.Log.Filename,
		MaxSize:    configs.Conf.Log.MaxSize,
		MaxAge:     configs.Conf.Log.MaxAge,
		MaxBackups: configs.Conf.Log.MaxBackups,
	}
	if err := log.SetupLog(c.Context, svcLog, configs.Conf.Log.SentryDSN); err != nil {
		zerolog.Fatal().Err(err).Msg("failed to setup log")
		return err
	}
	// log config
	dump, err := configs.Conf.Dump()
	if err != nil {
		log.Error(c.Context, err, "failed to dump config")
		return errors.Wrap(err, "")
	}
	log.Infof(c.Context, "%s", dump)

	// wait for unix signals and try to GracefulStop
	ctx, cancel := signal.NotifyContext(c.Context, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	br, err := boar.New(ctx, &configs.Conf, nil)
	if err != nil {
		log.Error(c.Context, err, "failed to create boar")
		return err
	}
	defer br.Close()

	if err := virt.Cleanup(); err != nil {
		return errors.Wrap(err, "")
	}

	grpcSrv, err := grpcserver.New(&configs.Conf, br)
	if err != nil {
		log.Error(c.Context, err, "failed to create grpc server")
		return err
	}

	errExitCh := make(chan struct{})
	if configs.Conf.BindHTTPAddr != "" {
		go startHTTPServer(configs.Conf.BindHTTPAddr)
	}
	go func() {
		defer close(errExitCh)
		if err := grpcSrv.Serve(); err != nil {
			log.Error(c.Context, err, "failed to start grpc server")
			metrics.IncrError()
		}
	}()
	log.Info(c.Context, "[main] all servers are running")

	select {
	case <-ctx.Done():
		log.Info(c.Context, "[main] interrupt by signal")
	case <-errExitCh:
		log.Warn(c.Context, "[main] server exit abnormally.")
	}

	grpcSrv.Stop(false)

	return nil
}
