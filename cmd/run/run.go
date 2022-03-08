package run

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/idgen"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/netx"
	"github.com/projecteru2/yavirt/store"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt"
	"github.com/projecteru2/yavirt/virt/guest/manager"
	"github.com/projecteru2/yavirt/vnet"
	"github.com/projecteru2/yavirt/vnet/calico"
	calinet "github.com/projecteru2/yavirt/vnet/calico"
	"github.com/projecteru2/yavirt/vnet/device"
	calihandler "github.com/projecteru2/yavirt/vnet/handler/calico"
)

var runtime Runtime

// Runner .
type Runner func(*cli.Context, Runtime) error

// Runtime .
type Runtime struct {
	ConfigFiles   []string
	SkipSetupHost bool
	Host          *model.Host
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
		if err := setup(); err != nil {
			return errors.Trace(err)
		}

		return fn(c, runtime)
	}
}

func setup() error {
	if len(runtime.ConfigFiles) > 0 {
		if err := config.Conf.Load(runtime.ConfigFiles); err != nil {
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
	hn, err := util.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	if runtime.Host, err = model.LoadHost(hn); err != nil {
		return errors.Annotatef(err, "invalid hostname %s", hn)
	}

	return nil
}

func setupCalico() (err error) {
	if endps := os.Getenv("ETCD_ENDPOINTS"); len(endps) < 1 {
		if err = os.Setenv("ETCD_ENDPOINTS", strings.Join(config.Conf.EtcdEndpoints, ",")); err != nil {
			return
		}
	}

	if runtime.Device, err = device.New(); err != nil {
		return
	}

	if runtime.CalicoDriver, err = calico.NewDriver(config.Conf.CalicoConfigFile, config.Conf.CalicoPoolNames); err != nil {
		return
	}

	var outboundIP string
	if outboundIP, err = netx.GetOutboundIP(config.Conf.CoreAddr); err != nil {
		return
	}

	runtime.CalicoHandler = calihandler.New(runtime.Device, runtime.CalicoDriver, config.Conf.CalicoPoolNames, outboundIP)
	err = runtime.CalicoHandler.InitGateway(config.Conf.CalicoGatewayName)

	return
}
