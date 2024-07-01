package boar

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/projecteru2/libyavirt/types"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/agent"
	"github.com/projecteru2/yavirt/internal/eru/recycle"
	"github.com/projecteru2/yavirt/internal/eru/resources"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	networkFactory "github.com/projecteru2/yavirt/internal/network/factory"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/utils/notify/bison"
	"github.com/projecteru2/yavirt/internal/ver"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/internal/vmcache"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/utils"
	vmiFact "github.com/yuyang0/vmimage/factory"
	vmitypes "github.com/yuyang0/vmimage/types"
)

// Boar .
type Boar struct {
	Host        *models.Host
	cfg         *configs.Config
	pool        *taskPool
	BootGuestCh chan<- string

	pid2ExitCode   *utils.ExitCodeMap
	RecoverGuestCh chan<- string

	watchers *interutils.Watchers

	imageMutex sync.Mutex
	agt        *agent.Manager
	mCol       *MetricsCollector
}

func New(ctx context.Context, cfg *configs.Config, t *testing.T) (br *Boar, err error) {
	var cols []prometheus.Collector

	br = &Boar{
		cfg:          cfg,
		mCol:         &MetricsCollector{},
		pid2ExitCode: utils.NewSyncMap(),
		watchers:     interutils.NewWatchers(),
	}
	// setup notify
	if err := bison.Setup(&cfg.Notify, t); err != nil {
		return br, errors.Wrap(err, "failed to setup notify")
	}

	resMgr, err := resources.Setup(ctx, cfg, t)
	if err != nil {
		return nil, err
	}
	cols = append(cols, resMgr.GetMetricsCollector())

	br.Host, err = models.LoadHost()
	if err != nil {
		return nil, err
	}
	br.pool, err = newTaskPool(cfg.MaxConcurrency)
	if err != nil {
		return nil, err
	}
	if err := idgen.Setup(br.Host.ID); err != nil {
		return nil, err
	}

	if err = store.Setup(configs.Conf, t); err != nil {
		return nil, err
	}
	go br.watchers.Run(ctx)
	if err := vmcache.Setup(ctx, cfg, br.watchers); err != nil {
		return br, errors.Wrap(err, "failed to setup vmcache")
	}
	if err := vmiFact.Setup(&cfg.ImageHub); err != nil {
		return br, errors.Wrap(err, "failed to setup vmimage")
	}
	if err := networkFactory.Setup(&cfg.Network); err != nil {
		return br, errors.Wrap(err, "failed to setup calico")
	}
	cols = append(cols, networkFactory.GetMetricsCollectors()...)

	if cfg.Eru.Enable {
		if err = recycle.Setup(ctx, &configs.Conf, t); err != nil {
			return br, errors.Wrap(err, "failed to setup recycle")
		}
		recycle.Run(ctx, br)

		parts := strings.Split(cfg.BindGRPCAddr, ":")
		if len(parts) != 2 {
			return br, errors.Newf("invalid bind addr %s", cfg.BindGRPCAddr)
		}
		grpcPort := parts[1]
		endpoint := fmt.Sprintf( //nolint
			"virt-grpc://%s:%s@%s:%s",
			cfg.Auth.Username, cfg.Auth.Password, cfg.Host.Addr, grpcPort,
		)
		br.agt, err = agent.NewManager(ctx, br, &cfg.Eru, endpoint, t)
		if err != nil {
			return br, errors.Wrap(err, "failed to setup agent")
		}
		go br.agt.Run(ctx) //nolint
		cols = append(cols, br.agt.GetMetricsCollector())
	}
	cols = append(cols, br.GetMetricsCollector())
	metrics.Setup(cfg.Host.Name, cols...)
	/*
		if err := svc.ScheduleSnapshotCreate(); err != nil {
			return errors.Wrap(err, "")
		}
	*/

	return br, nil
}

func (svc *Boar) Close() {
	_ = svc.agt.Exit()
	svc.pool.Release()
	store.Close()
}

// Ping .
func (svc *Boar) Ping() map[string]string {
	return map[string]string{"version": ver.Version()}
}

// Info .
func (svc *Boar) Info() (*types.HostInfo, error) {
	res, err := resources.GetManager().FetchResources()
	if err != nil {
		return nil, err
	}
	return &types.HostInfo{
		ID:        fmt.Sprintf("%d", svc.Host.ID),
		CPU:       svc.Host.CPU,
		Mem:       svc.Host.Memory,
		Storage:   svc.Host.Storage,
		Resources: res,
	}, nil
}

func (svc *Boar) IsHealthy(ctx context.Context) (ans bool) {
	logger := log.WithFunc("boar.Healthy")
	var err error
	// check image service
	if err1 := vmiFact.CheckHealth(ctx); err1 != nil {
		svc.mCol.imageHealthy.Store(false)
		err = errors.CombineErrors(err, errors.WithMessagef(err1, "failed to check image hub")) //nolint
		return false
	}
	svc.mCol.imageHealthy.Store(true)
	// check libvirt health
	if err1 := checkLibvirtSocket(); err1 != nil {
		svc.mCol.libvirtHealthy.Store(false)
		err = errors.CombineErrors(err, errors.WithMessagef(err1, "failed to check libvirt socket"))
	}
	svc.mCol.libvirtHealthy.Store(true)
	// check network drivers, including clico, ovn etc
	if err1 := networkFactory.CheckHealth(ctx); err1 != nil {
		err = errors.CombineErrors(err, errors.WithMessagef(err1, "failed to check network drivers"))
	}
	//TODO:Check more things

	if err != nil {
		logger.Errorf(ctx, err, "failed to check health")
		return false
	}
	return true
}

// GetGuest .
func (svc *Boar) GetGuest(ctx context.Context, id string) (*types.Guest, error) {
	vg, err := svc.loadGuest(ctx, id)
	if err != nil {
		log.WithFunc("boar.GetGuest").Error(ctx, err)
		metrics.IncrError()
		return nil, err
	}
	resp := convGuestResp(vg.Guest)
	return resp, nil
}

// GetGuestIDList .
func (svc *Boar) GetGuestIDList(ctx context.Context) ([]string, error) {
	ids, err := svc.ListLocalIDs(ctx, true)
	if err != nil {
		log.WithFunc("boar.GetGuestIDList").Error(ctx, err)
		metrics.IncrError()
		return nil, err
	}
	return ids, nil
}

// GetGuestUUID .
func (svc *Boar) GetGuestUUID(ctx context.Context, id string) (string, error) {
	uuid, err := svc.LoadUUID(ctx, id)
	if err != nil {
		log.WithFunc("boar.GetGuestUUID").Error(ctx, err)
		metrics.IncrError()
		return "", err
	}
	return uuid, nil
}

// CaptureGuest .
func (svc *Boar) CaptureGuest(ctx context.Context, id string, imgName string, overridden bool) (uimg *vmitypes.Image, err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return nil, err
	}
	uImg, err := g.Capture(imgName, overridden)
	if err != nil {
		return nil, err
	}
	return uImg, nil
}

// ResizeGuest re-allocates spec or volumes.
func (svc *Boar) ResizeGuest(ctx context.Context, id string, opts *intertypes.GuestResizeOption) (err error) {
	defer logErr(err)

	vols, err := extractVols(opts.Resources)
	if err != nil {
		return err
	}
	cpumem, err := extractCPUMem(opts.Resources)
	if err != nil {
		return err
	}
	gpu, err := extractGPU(opts.Resources)
	if err != nil {
		return err
	}
	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return err
	}
	do := func(_ context.Context) (any, error) {
		return nil, g.Resize(cpumem, gpu, vols)
	}
	_, err = svc.do(ctx, id, intertypes.ResizeOp, do, nil)
	return
}

// Wait .
func (svc *Boar) Wait(ctx context.Context, id string, block bool) (msg string, code int, err error) {
	defer logErr(err)

	err = svc.stopGuest(ctx, id, !block)
	if err != nil {
		return "stop error", -1, err
	}

	err = svc.ctrl(ctx, id, intertypes.MiscOp, func(g *guest.Guest) error {
		if err = g.Wait(meta.StatusStopped, block); err != nil {
			return err
		}

		if g.LambdaOption != nil {
			msg = string(g.LambdaOption.CmdOutput)
			code = g.LambdaOption.ExitCode
		}

		return nil
	}, nil)
	return msg, code, err
}

// ListLocals lists all local guests.
func (svc *Boar) ListLocalIDs(ctx context.Context, onlyERU bool) ([]string, error) {
	ids, err := guest.ListLocalIDs(ctx)
	if err != nil {
		return nil, err
	}
	if !onlyERU {
		return ids, nil
	}
	var ans []string
	for _, id := range ids {
		if idgen.CheckID(id) {
			ans = append(ans, id)
		}
	}
	return ans, nil
}

// LoadUUID read a guest's UUID.
func (svc *Boar) LoadUUID(ctx context.Context, id string) (string, error) {
	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return "", err
	}
	return g.GetUUID()
}

// loadGuest read a guest from metadata.
func (svc *Boar) loadGuest(ctx context.Context, id string, opts ...models.Option) (*guest.Guest, error) {
	g, err := models.LoadGuest(id)
	if err != nil {
		return nil, err
	}

	var vg = guest.New(ctx, g)
	if err := vg.Load(opts...); err != nil {
		return nil, err
	}
	if err = vg.UpdateStateIfNecessary(); err != nil {
		log.WithFunc("boar.loadGuest").Warnf(ctx, "update state error: %s", err)
	}
	return vg, nil
}

func (svc *Boar) WatchGuestEvents(context.Context) (*interutils.Watcher, error) {
	return svc.watchers.Get()
}

func logErr(err error) {
	if err != nil {
		log.Error(context.TODO(), err)
		metrics.IncrError()
	}
}

type ctrlFunc func(*guest.Guest) error
type rollbackFunc func()

func (svc *Boar) ctrl(ctx context.Context, id string, op intertypes.Operator, fn ctrlFunc, rollback rollbackFunc) error { //nolint
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, err
		}
		return nil, fn(g)
	}
	_, err := svc.do(ctx, id, op, do, rollback)
	return err
}

type doFunc func(context.Context) (any, error)

func (svc *Boar) do(ctx context.Context, id string, op intertypes.Operator, fn doFunc, rollback rollbackFunc) (result any, err error) {
	defer func() {
		if err != nil && rollback != nil {
			rollback()
		}
	}()

	// add a max timeout
	ctx1, cancel := context.WithTimeout(ctx, configs.Conf.VirtTimeout)
	defer cancel()

	t := newTask(ctx1, id, op, fn)

	if err := svc.pool.SubmitTask(t); err != nil {
		return nil, err
	}

	metrics.Incr(metrics.MetricSvcTasks, nil)       //nolint:errcheck
	defer metrics.Decr(metrics.MetricSvcTasks, nil) //nolint:errcheck

	select {
	case <-t.Done():
		result, err = t.result()
	case <-ctx1.Done():
		err = ctx1.Err()
	}
	if err != nil {
		metrics.IncrError()
		return
	}

	svc.watchers.Watched(intertypes.Event{
		ID:   id,
		Type: guestEventType,
		Op:   op,
		Time: time.Now().UTC(),
	})

	return
}

const guestEventType = "guest"

func checkLibvirtSocket() error {
	socketPath := "/var/run/libvirt/libvirt-sock"
	// Dial the Unix domain socket
	conn, err := net.DialTimeout("unix", socketPath, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
