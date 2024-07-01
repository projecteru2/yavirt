package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/projecteru2/yavirt/internal/eru/common"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/utils"

	"github.com/projecteru2/core/cluster"
	"github.com/projecteru2/core/log"
	yavirttypes "github.com/projecteru2/libyavirt/types"
)

func (m *Manager) ListWorkloadIDs(ctx context.Context) (ids []string, err error) {
	utils.WithTimeout(ctx, m.config.GlobalConnectionTimeout, func(ctx context.Context) {
		ids, err = m.svc.GetGuestIDList(ctx)
	})
	if err != nil && !strings.Contains(err.Error(), "key not exists") {
		log.WithFunc("ListWorkloadIDs").Error(ctx, err, "failed to get workload ids")
		return nil, err
	}
	return ids, nil
}

func (m *Manager) detectWorkload(ctx context.Context, ID string) (*Guest, error) {
	logger := log.WithFunc("detectWorkload").WithField("ID", ID)

	var guest *yavirttypes.Guest
	var err error

	utils.WithTimeout(ctx, m.config.GlobalConnectionTimeout, func(ctx context.Context) {
		guest, err = m.svc.GetGuest(ctx, ID)
	})

	if err != nil {
		logger.Error(ctx, err, "failed to detect workload")
		return nil, err
	}

	if _, ok := guest.Labels[cluster.ERUMark]; !ok {
		return nil, common.ErrInvaildVM
	}

	if m.config.CheckOnlyMine && m.config.Hostname != guest.Hostname {
		logger.Debugf(ctx, "guest's hostname is %s instead of %s", guest.Hostname, m.config.Hostname)
		return nil, common.ErrInvaildVM
	}

	return &Guest{
		ID:            guest.ID,
		Status:        guest.Status,
		TransitStatus: guest.TransitStatus,
		CreateTime:    guest.CreateTime,
		TransitTime:   guest.TransitTime,
		UpdateTime:    guest.UpdateTime,
		CPU:           guest.CPU,
		Mem:           guest.Mem,
		Storage:       guest.Storage,
		ImageID:       guest.ImageID,
		ImageName:     guest.ImageName,
		ImageUser:     guest.ImageUser,
		Networks:      guest.Networks,
		Labels:        guest.Labels,
		IPs:           guest.IPs,
		Hostname:      guest.Hostname,
		Running:       guest.Running,
		once:          sync.Once{},
	}, nil
}

// GetStatus checks workload's status first, then returns workload status
func (m *Manager) GetStatus(ctx context.Context, ID string, checkHealth bool) (*types.WorkloadStatus, error) {
	logger := log.WithFunc("GetStatus").WithField("ID", ID)
	guest, err := m.detectWorkload(ctx, ID)
	if err != nil {
		logger.Error(ctx, err, "failed to get guest status")
		return nil, err
	}

	bytes, err := json.Marshal(guest.Labels)
	if err != nil {
		logger.Error(ctx, err, "failed to marshal labels")
		return nil, err
	}

	status := &types.WorkloadStatus{
		ID:        guest.ID,
		Running:   guest.Running,
		Healthy:   guest.Running && guest.HealthCheck == nil,
		Networks:  guest.Networks,
		Extension: bytes,
		Nodename:  m.config.Hostname,
	}

	if checkHealth && guest.Running {
		free, acquired := m.cas.Acquire(guest.ID)
		if !acquired {
			return nil, common.ErrGetLockFailed
		}
		defer free()
		timeout := time.Duration(m.config.HealthCheck.Timeout) * time.Second
		status.Healthy = guest.CheckHealth(ctx, m.svc, timeout, m.config.HealthCheck.EnableDefaultChecker)
	}

	return status, nil
}

func (m *Manager) healthCheck(ctx context.Context) {
	tick := time.NewTicker(time.Duration(m.config.HealthCheck.Interval) * time.Second)
	defer tick.Stop()

	_ = utils.Pool.Submit(func() { m.checkAllWorkloads(ctx) })

	for {
		select {
		case <-tick.C:
			_ = utils.Pool.Submit(func() { m.checkAllWorkloads(ctx) })
		case <-ctx.Done():
			return
		}
	}
}

// 检查全部 label 为ERU=1的workload
// 这里需要 list all，原因是 monitor 检测到 die 的时候已经标记为 false 了
// 但是这时候 health check 刚返回 true 回来并写入 core
// 为了保证最终数据一致性这里也要检测
func (m *Manager) checkAllWorkloads(ctx context.Context) {
	logger := log.WithFunc("checkAllWorkloads")
	logger.Debug(ctx, "health check begin")
	workloadIDs, err := m.ListWorkloadIDs(ctx)
	if err != nil {
		logger.Error(ctx, err, "error when list all workloads with label \"ERU=1\"")
		return
	}

	for _, workloadID := range workloadIDs {
		ID := workloadID
		_ = utils.Pool.Submit(func() { m.checkOneWorkload(ctx, ID) })
	}
}

// 检查并保存一个workload的状态，最后返回workload是否healthy。
// 返回healthy是为了重试用的，没啥别的意义。
func (m *Manager) checkOneWorkload(ctx context.Context, ID string) bool {
	logger := log.WithFunc("checkOneWorkload").WithField("ID", ID)
	workloadStatus, err := m.GetStatus(ctx, ID, true)
	if err != nil {
		logger.Error(ctx, err, "failed to get status of workload")
		return false
	}

	m.wrkStatusCache.Set(workloadStatus.ID, workloadStatus, 0)

	if err = m.setWorkloadStatus(ctx, workloadStatus); err != nil {
		logger.Error(ctx, err, "update workload status failed")
	}
	return workloadStatus.Healthy
}

// 设置workload状态，允许重试，带timeout控制
func (m *Manager) setWorkloadStatus(ctx context.Context, status *types.WorkloadStatus) error {
	return utils.BackoffRetry(ctx, 3, func() error {
		return m.store.SetWorkloadStatus(ctx, status, m.config.GetHealthCheckStatusTTL())
	})
}
