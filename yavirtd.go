package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/server"
	grpcserver "github.com/projecteru2/yavirt/internal/server/grpc"
	httpserver "github.com/projecteru2/yavirt/internal/server/http"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/store"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	cli.VersionPrinter = func(c *cli.Context) {
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
				Value:   "INFO",
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
		log.ErrorStack(err)
		metrics.IncrError()
		os.Exit(1)
	}

	os.Exit(0)
}
func initConfig(c *cli.Context) error {
	cfg := &configs.Conf
	if err := cfg.Load([]string{c.String("config")}); err != nil {
		return err
	}
	return cfg.Prepare(c)
}

// Run .
func Run(c *cli.Context) error {
	if err := initConfig(c); err != nil {
		return err
	}
	defers, svc, err := setup()
	if err != nil {
		return errors.Trace(err)
	}
	defer defers()
	defer store.Close()

	dump, err := configs.Conf.Dump()
	if err != nil {
		return errors.Trace(err)
	}
	log.Infof("%s", dump)

	if err := virt.Cleanup(); err != nil {
		return errors.Trace(err)
	}

	// setup epoller
	if err := guest.SetupEpoller(); err != nil {
		return errors.Trace(err)
	}
	defer guest.GetCurrentEpoller().Close()

	grpcSrv, err := grpcserver.Listen(svc)
	if err != nil {
		return errors.Trace(err)
	}

	httpSrv, err := httpserver.Listen(svc)
	if err != nil {
		return errors.Trace(err)
	}

	go prof(configs.Conf.ProfHTTPPort)

	run([]server.Serverable{grpcSrv, httpSrv})

	return nil
}

func run(servers []server.Serverable) {
	defer log.Warnf("[main] yavirtd proc exit")

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)

		go handleSigns(srv)

		go func(server server.Serverable) {
			defer wg.Done()
			if err := server.Serve(); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
			}
		}(srv)
	}

	log.Infof("[main] all servers're running")

	go notify(servers)

	wg.Wait()
}

func notify(servers []server.Serverable) {
	defer log.Infof("[main] exit notify loop exit")

	var wg sync.WaitGroup

	for _, srv := range servers {
		wg.Add(1)

		go func(exitCh <-chan struct{}) {
			defer func() {
				log.Infof("[main] server exitCh %p monitor was stopped", exitCh)
				wg.Done()
			}()

			select {
			case <-exitCh:
				log.Infof("[main] recv from server exit ch %p", exitCh)
				closeExitNoti()

			case <-exitNoti:
				log.Infof("[main] recv exit notification: %p", exitCh)
			}
		}(srv.ExitCh())
	}

	wg.Wait()
}

func setup() (deferSentry func(), svc *server.Service, err error) {
	if deferSentry, err = log.Setup(configs.Conf.LogLevel, configs.Conf.LogFile, configs.Conf.LogSentry); err != nil {
		return
	}

	if err = store.Setup(configs.Conf.MetaType); err != nil {
		return
	}

	if svc, err = server.SetupYavirtdService(); err != nil {
		return
	}

	idgen.Setup(svc.Host.ID, time.Now())

	models.Setup()

	return
}

var signs = []os.Signal{
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
	syscall.SIGUSR2,
}

func handleSigns(srv server.Serverable) {
	defer func() {
		log.Warnf("[main] signal handler for %p exit", srv)
		srv.Close()
	}()

	var signCh = make(chan os.Signal, 1)
	signal.Notify(signCh, signs...)

	var exit = srv.ExitCh()

	for {
		select {
		case sign := <-signCh:
			switch sign {
			case syscall.SIGUSR2:
				log.Warnf("[main] got sign USR2 to reload")
				log.ErrorStack(srv.Reload())

			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Warnf("[main] got sign %d to exit", sign)
				return

			default:
				log.Warnf("[main] got sign %d to ignore", sign)
			}

		case <-exitNoti:
			log.Warnf("[main] recv an exit notification: %p", srv)
			return

		case <-exit:
			log.Warnf("[main] recv from server %p exit ch", srv)
			return
		}
	}
}

var (
	exitNoti     = make(chan struct{}, 1)
	exitNotiOnce sync.Once
)

func closeExitNoti() {
	exitNotiOnce.Do(func() {
		close(exitNoti)
	})
}

func prof(port int) {
	var enable = strings.ToLower(os.Getenv("YAVIRTD_PPROF"))
	switch enable {
	case "":
		fallthrough
	case "0":
		fallthrough
	case "false":
		fallthrough
	case "off":
		http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil) //nolint
	default:
		return
	}
}
