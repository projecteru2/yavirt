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

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/store"
	"github.com/projecteru2/yavirt/ver"
	"github.com/projecteru2/yavirt/virt"
	"github.com/projecteru2/yavirt/virt/guest"
	"github.com/projecteru2/yavirt/yavirtd"
	grpcserver "github.com/projecteru2/yavirt/yavirtd/grpc"
	httpserver "github.com/projecteru2/yavirt/yavirtd/http"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2) //nolint

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(ver.Version())
	}

	app := &cli.App{
		Name:    "yavirtd",
		Usage:   "yavirt daemon",
		Version: "v",
		Action:  Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.ErrorStack(err)
		metric.IncrError()
		os.Exit(1)
	}

	os.Exit(0)
}

// Run .
func Run(c *cli.Context) error {
	defers, svc, err := setup(c.Args().Slice())
	if err != nil {
		return errors.Trace(err)
	}
	defer defers()
	defer store.Close()

	dump, err := config.Conf.Dump()
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
	defer guest.GetCurrentEpoller().Close() // nolint

	grpcSrv, err := grpcserver.Listen(svc)
	if err != nil {
		return errors.Trace(err)
	}

	httpSrv, err := httpserver.Listen(svc)
	if err != nil {
		return errors.Trace(err)
	}

	go prof(config.Conf.ProfHTTPPort)

	run([]yavirtd.Server{grpcSrv, httpSrv})

	return nil
}

func run(servers []yavirtd.Server) {
	defer log.Warnf("[main] yavirtd proc exit")

	var wg sync.WaitGroup
	for _, srv := range servers {
		wg.Add(1)

		go handleSigns(srv)

		go func(server yavirtd.Server) {
			defer wg.Done()
			if err := server.Serve(); err != nil {
				log.ErrorStack(err)
				metric.IncrError()
			}
		}(srv)
	}

	log.Infof("[main] all servers're running")

	go notify(servers)

	wg.Wait()
}

func notify(servers []yavirtd.Server) {
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

func setup(configFiles []string) (deferSentry func(), svc *yavirtd.Service, err error) {
	if err = config.Conf.Load(configFiles); err != nil {
		return
	}

	if deferSentry, err = log.Setup(config.Conf.LogLevel, config.Conf.LogFile, config.Conf.LogSentry); err != nil {
		return
	}

	if err = store.Setup(config.Conf.MetaType); err != nil {
		return
	}

	if svc, err = yavirtd.SetupYavirtdService(); err != nil {
		return
	}

	idgen.Setup(svc.Host.ID, time.Now())

	model.Setup()

	return
}

var signs = []os.Signal{
	syscall.SIGHUP,
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGQUIT,
	syscall.SIGUSR2,
}

func handleSigns(srv yavirtd.Server) {
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
