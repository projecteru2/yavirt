package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alphadose/haxmap"
	"github.com/cenkalti/backoff/v4"
	"github.com/cockroachdb/errors"
	"github.com/patrickmn/go-cache"
	"github.com/projecteru2/core/log"
	corerpc "github.com/projecteru2/core/rpc"
	"github.com/projecteru2/yavirt/internal/eru/common"
	"github.com/projecteru2/yavirt/internal/eru/store"
	corestore "github.com/projecteru2/yavirt/internal/eru/store/core"
	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/internal/utils"
	"google.golang.org/grpc/status"
)

type Manager struct {
	svc    service.Service
	store  store.Store
	config *types.Config

	checkWorkloadMutex *sync.Mutex
	startingWorkloads  *haxmap.Map[string, *utils.RetryTask]

	// storeIdentifier indicates which eru this agent belongs to
	// it can be used to identify the corresponding core
	// and all containers that belong to this core
	storeIdentifier string
	cas             *utils.GroupCAS
	wrkStatusCache  *cache.Cache

	mCol *MetricsCollector
}

func NewManager(
	ctx context.Context,
	svc service.Service,
	config *types.Config,
	eruEndpoint string,
	t *testing.T,
) (*Manager, error) {
	logger := log.WithFunc("agent.NewManager")
	interval := time.Duration(2*config.HealthCheck.Interval) * time.Second
	m := &Manager{
		config:             config,
		svc:                svc,
		cas:                utils.NewGroupCAS(),
		checkWorkloadMutex: &sync.Mutex{},
		startingWorkloads:  haxmap.New[string, *utils.RetryTask](),
		wrkStatusCache:     cache.New(interval, interval),
	}
	m.mCol = &MetricsCollector{
		wrkStatusCache: m.wrkStatusCache,
	}

	if t == nil {
		corestore.Init(ctx, config)
		if m.store = corestore.Get(); m.store == nil {
			return nil, common.ErrGetStoreFailed
		}
	} else {
		m.store = storemocks.NewFakeStore()
	}
	labels := map[string]string{}
	for _, label := range config.Labels {
		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return nil, errors.Newf("invalid label %s", label)
		}
		labels[parts[0]] = parts[1]
	}
	go func() {
		if err := m.addCurrentNodeWithRetry(ctx, config, eruEndpoint, labels); err != nil {
			logger.Errorf(ctx, err, "failed to register current node")
			return
		}
		// update node's labels if necessary
		if len(labels) > 0 {
			if _, err := m.store.SetNode(ctx, &types.SetNodeOpts{
				Nodename: config.Hostname,
				Labels:   labels,
			}); err != nil {
				logger.Errorf(ctx, err, "failed to update node labels")
			}
		}
	}()
	m.storeIdentifier = m.store.GetIdentifier(ctx)
	return m, nil
}

func (m *Manager) startNodeManager(ctx context.Context) {
	log.WithFunc("startNodeManager").Info(ctx, "starting node status heartbeat")
	_ = utils.Pool.Submit(func() { m.heartbeat(ctx) })
}

func (m *Manager) startWorkloadManager(ctx context.Context) {
	log.WithFunc("startWorkloadManager").Info(ctx, "starting workload manager")
	// start status watcher
	_ = utils.Pool.Submit(func() { m.monitor(ctx) })

	// start health check
	_ = utils.Pool.Submit(func() { m.healthCheck(ctx) })
}

// Run runs a node manager
func (m *Manager) Run(ctx context.Context) error {
	logger := log.WithFunc("Run")

	m.startNodeManager(ctx)
	m.startWorkloadManager(ctx)

	<-ctx.Done()
	logger.Info(ctx, "exiting")
	return nil
}

// Exit .
func (m *Manager) Exit() error {
	ctx := context.TODO()
	logger := log.WithFunc("Exit").WithField("hostname", m.config.Hostname)
	logger.Info(ctx, "remove node status")

	// ctx is now canceled. use a new context.
	var err error
	utils.WithTimeout(ctx, m.config.GlobalConnectionTimeout, func(ctx context.Context) {
		// remove node status
		err = m.store.SetNodeStatus(ctx, -1)
	})
	if err != nil {
		logger.Error(ctx, err, "failed to remove node status")
		return err
	}
	return nil
}

func (m *Manager) addCurrentNodeWithRetry(
	ctx context.Context,
	config *types.Config,
	eruEndpoint string,
	labels map[string]string,
) error {
	logger := log.WithFunc("addCurrentNodeWithRetry").WithField("hostname", config.Hostname)
	interval := 10 * time.Second
	maxRetries := 10
	time.Sleep(interval)
	bf := backoff.NewConstantBackOff(interval)
	return backoff.Retry(func() error {
		// try to register current node to eru core
		if _, err := m.store.AddNode(ctx, &types.AddNodeOpts{
			Nodename: config.Hostname,
			Endpoint: eruEndpoint,
			Podname:  config.Podname,
			Labels:   labels,
		}); err != nil {
			e, ok := status.FromError(err)
			if !ok {
				logger.Error(ctx, err, "failed to add node")
				return err
			}
			if e.Code() == corerpc.AddNode && strings.Contains(e.Message(), "node already exists") {
				logger.Infof(ctx, "node %s already exists", config.Hostname)
				return nil
			}
			logger.Errorf(ctx, err, "failed to add node %s", config.Hostname)
			return err
		}
		return nil
	}, backoff.WithMaxRetries(bf, uint64(maxRetries)))
}
