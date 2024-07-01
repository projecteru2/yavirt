package agent

import (
	"context"
	"time"

	"github.com/projecteru2/yavirt/internal/eru/common"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/utils"

	"github.com/projecteru2/core/log"
)

func (m *Manager) initMonitor(ctx context.Context) (err error) {
	watcher, err := m.svc.WatchGuestEvents(ctx)
	if err != nil {
		return err
	}
	logger := log.WithFunc("initMonitor")
	defer logger.Infof(ctx, "events goroutine has done")
	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.Events():
			// don't block here
			_ = utils.Pool.Submit(func() {
				switch event.Op {
				case intertypes.StartOp:
					m.handleWorkloadStart(ctx, event.ID)
				case intertypes.DieOp:
					m.handleWorkloadDie(ctx, event.ID)
				}
			})
		case <-watcher.Done():
			// The watcher already has been stopped.
			logger.Infof(ctx, "watcher has done")
			return nil

		case <-ctx.Done():
			logger.Infof(ctx, "ctx done")
			return nil
		}
	}
}

// monitor with retry
func (m *Manager) monitor(ctx context.Context) {
	logger := log.WithFunc("monitor")
	for {
		select {
		case <-ctx.Done():
			logger.Info(ctx, "context canceled, stop monitoring")
			return
		default:
			if err := m.initMonitor(ctx); err != nil {
				logger.Error(ctx, err, "received an err, will retry")
			}
			time.Sleep(m.config.GlobalConnectionTimeout)
		}
	}
}

// 检查一个workload，允许重试
func (m *Manager) checkOneWorkloadWithBackoffRetry(ctx context.Context, ID string) {
	logger := log.WithFunc("checkOneWorkloadWithBackoffRetry").WithField("ID", ID)
	logger.Debug(ctx, "check workload")

	m.checkWorkloadMutex.Lock()
	defer m.checkWorkloadMutex.Unlock()

	if retryTask, ok := m.startingWorkloads.Get(ID); ok {
		retryTask.Stop(ctx)
	}

	retryTask := utils.NewRetryTask(ctx, utils.GetMaxAttemptsByTTL(m.config.GetHealthCheckStatusTTL()), func() error {
		if !m.checkOneWorkload(ctx, ID) {
			// 这个err就是用来判断要不要继续的，不用打在日志里
			return common.ErrWorkloadUnhealthy
		}
		return nil
	})
	m.startingWorkloads.Set(ID, retryTask)
	_ = utils.Pool.Submit(func() {
		if err := retryTask.Run(ctx); err != nil {
			logger.Debug(ctx, "workload still not healthy")
		}
	})
}

func (m *Manager) handleWorkloadStart(ctx context.Context, ID string) {
	logger := log.WithFunc("handleWorkloadStart").WithField("ID", ID)
	logger.Debug(ctx, "workload start")
	workloadStatus, err := m.GetStatus(ctx, ID, true)
	if err != nil {
		logger.Warnf(ctx, "faild to get workload status: %s", err)
		return
	}

	if workloadStatus.Healthy {
		if err := m.store.SetWorkloadStatus(ctx, workloadStatus, m.config.GetHealthCheckStatusTTL()); err != nil {
			logger.Warnf(ctx, "failed to update deploy status: %s", err)
		}
	} else {
		m.checkOneWorkloadWithBackoffRetry(ctx, ID)
	}
}

func (m *Manager) handleWorkloadDie(ctx context.Context, ID string) {
	logger := log.WithFunc("handleWorkloadDie").WithField("ID", ID)
	logger.Debug(ctx, "wrokload die")
	workloadStatus, err := m.GetStatus(ctx, ID, true)
	if err != nil {
		logger.Warnf(ctx, "faild to get workload status: %s", err)
		return
	}

	if err := m.store.SetWorkloadStatus(ctx, workloadStatus, m.config.GetHealthCheckStatusTTL()); err != nil {
		logger.Warnf(ctx, "failed to update deploy status: %s", err)
	}
}
