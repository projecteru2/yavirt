package agent

import (
	"context"
	"time"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/utils"
)

// heartbeat creates a new goroutine to report status every HeartbeatInterval seconds
// By default HeartbeatInterval is 0, will not do heartbeat.
func (m *Manager) heartbeat(ctx context.Context) {
	if m.config.HeartbeatInterval <= 0 {
		return
	}
	_ = utils.Pool.Submit(func() { m.nodeStatusReport(ctx) })

	tick := time.NewTicker(time.Duration(m.config.HeartbeatInterval) * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			_ = utils.Pool.Submit(func() { m.nodeStatusReport(ctx) })
		case <-ctx.Done():
			return
		}
	}
}

// nodeStatusReport does heartbeat, tells core this node is alive.
// The TTL is set to double of HeartbeatInterval, by default it will be 360s,
// which means if a node is not available, subcriber will notice this after at least 360s.
// HealthCheck.Timeout is used as timeout of requesting core Profile
func (m *Manager) nodeStatusReport(ctx context.Context) {
	logger := log.WithFunc("nodeStatusReport").WithField("hostname", m.config.Hostname)
	logger.Debug(ctx, "report begins")
	defer logger.Debug(ctx, "report ends")
	if !m.svc.IsHealthy(ctx) {
		logger.Warn(ctx, "service is not healthy")
		return
	}
	if err := m.store.CheckHealth(ctx); err != nil {
		m.mCol.coreHealthy.Store(false)
	} else {
		m.mCol.coreHealthy.Store(true)
	}
	ttl := int64(m.config.HeartbeatInterval * 3)

	if err := utils.BackoffRetry(ctx, 3, func() (err error) {
		utils.WithTimeout(ctx, m.config.GlobalConnectionTimeout, func(ctx context.Context) {
			if err = m.store.SetNodeStatus(ctx, ttl); err != nil {
				logger.Error(ctx, err, "failed to set node status")
			}
		})
		return err
	}); err != nil {
		logger.Error(ctx, err, "failed to set node status for 3 times")
	}
}
