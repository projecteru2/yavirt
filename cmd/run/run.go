package run

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/internal/service/boar"
	"github.com/projecteru2/yavirt/pkg/netx"
)

var runtime Runtime

// Runner .
type Runner func(*cli.Context, Runtime) error

// Runtime .
type Runtime struct {
	Ctx      context.Context
	CancelFn context.CancelFunc
	Svc      service.Service
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
	return func(c *cli.Context) (err error) {
		cfg := &configs.Conf

		if err := cfg.Load(c.String("config")); err != nil {
			return errors.Wrap(err, "")
		}
		if err := cfg.Prepare(c); err != nil {
			return err
		}

		// always send log to stdout
		svcLog := &coretypes.ServerLogConfig{
			Level:      configs.Conf.Log.Level,
			UseJSON:    configs.Conf.Log.UseJSON,
			Filename:   configs.Conf.Log.Filename,
			MaxSize:    configs.Conf.Log.MaxSize,
			MaxAge:     configs.Conf.Log.MaxAge,
			MaxBackups: configs.Conf.Log.MaxBackups,
		}
		if err := log.SetupLog(c.Context, svcLog, configs.Conf.Log.SentryDSN); err != nil {
			return err
		}

		runtime.Ctx, runtime.CancelFn = context.WithTimeout(context.Background(), time.Duration(c.Int("timeout"))*time.Second)
		runtime.Svc, err = boar.New(c.Context, &configs.Conf, nil)
		if err != nil {
			return errors.Wrap(err, "")
		}

		return fn(c, runtime)
	}
}
