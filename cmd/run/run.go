package run

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/internal/virt/guest/manager"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/calico"
	calinet "github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/utils"
)

var runtime Runtime

// Runner .
type Runner func(*cli.Context, Runtime) error

// Runtime .
type Runtime struct {
	ConfigFiles   []string
	SkipSetupHost bool
	Host          *models.Host
	Device        *device.Driver
	CalicoDriver  *calinet.Driver
	CalicoHandler *calihandler.Handler
	Guest         manager.Manager
}

// VirtContext .
func (r Runtime) VirtContext() virt.Context {
	return virt.NewContext(context.Background(), r.CalicoHandler)
}

// ConvDecimal .
func (r Runtime) ConvDecimal(ipv4 string) int64 {
	if len(ipv4) < 1 {
		return 0
	}

	var dec, err = netx.IPv4ToInt(ipv4)
	if err != nil {
		panic(err)
	}

	return dec
}

// Run .
func Run(fn Runner) cli.ActionFunc {
	return func(c *cli.Context) error {
		runtime.ConfigFiles = c.StringSlice("config")
		runtime.SkipSetupHost = c.Bool("skip-setup-host")
		runtime.Guest = manager.New()
		// when add host, we need skip host setup
		if c.Command.FullName() == "host add" {
			runtime.SkipSetupHost = true
		}
		if err := setup(); err != nil {
			return errors.Trace(err)
		}

		return fn(c, runtime)
	}
}

func setup() error {
	if len(runtime.ConfigFiles) > 0 {
		if err := configs.Conf.Load(runtime.ConfigFiles); err != nil {
			return errors.Trace(err)
		}
	}

	if err := store.Setup("etcd"); err != nil {
		return errors.Trace(err)
	}

	if runtime.SkipSetupHost {
		return nil
	}

	if err := setupHost(); err != nil {
		return errors.Trace(err)
	}

	idgen.Setup(runtime.Host.ID, time.Now())

	if runtime.Host.NetworkMode == vnet.NetworkCalico {
		if err := setupCalico(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func setupHost() error {
	hn, err := utils.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	if runtime.Host, err = models.LoadHost(hn); err != nil {
		return errors.Annotatef(err, "invalid hostname %s", hn)
	}

	return nil
}

func setupCalico() (err error) {
	if endps := os.Getenv("ETCD_ENDPOINTS"); len(endps) < 1 {
		if err = os.Setenv("ETCD_ENDPOINTS", strings.Join(configs.Conf.EtcdEndpoints, ",")); err != nil {
			return
		}
	}

	if runtime.Device, err = device.New(); err != nil {
		return
	}

	if runtime.CalicoDriver, err = calico.NewDriver(configs.Conf.CalicoConfigFile, configs.Conf.CalicoPoolNames); err != nil {
		return
	}

	var outboundIP string
	if outboundIP, err = netx.GetOutboundIP(configs.Conf.CoreAddr); err != nil {
		return
	}

	runtime.CalicoHandler = calihandler.New(runtime.Device, runtime.CalicoDriver, configs.Conf.CalicoPoolNames, outboundIP)
	err = runtime.CalicoHandler.InitGateway(configs.Conf.CalicoGatewayName)

	return
}
