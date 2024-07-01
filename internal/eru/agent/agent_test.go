package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/service/mocks"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/pkg/test/mock"

	"github.com/stretchr/testify/assert"
)

func newMockManager(t *testing.T) *Manager {
	config := &types.Config{
		Hostname:          "fake",
		HeartbeatInterval: 2,
		CheckOnlyMine:     false,
		HealthCheck: types.HealthCheckConfig{
			Interval: 10,
			Timeout:  5,
			CacheTTL: 300,
		},
		GlobalConnectionTimeout: 5 * time.Second,
	}
	svc := &mocks.Service{}

	m, err := NewManager(context.Background(), svc, config, "", t)
	assert.Nil(t, err)
	return m
}

func TestRunNodeManager(t *testing.T) {
	manager := newMockManager(t)
	store := manager.store.(*storemocks.MockStore)
	svc := manager.svc.(*mocks.Service)
	svc.On("IsHealthy", mock.Anything).Return(true)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(manager.config.HeartbeatInterval*3)*time.Second)
	defer cancel()

	status, err := store.GetNodeStatus(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, status.Alive, false)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Duration(manager.config.HeartbeatInterval*2) * time.Second)
		status, err := store.GetNodeStatus(ctx, "fake")
		assert.Nil(t, err)
		assert.Equal(t, status.Alive, true)
	}()

	manager.startNodeManager(ctx)

	info, err := store.GetNode(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, info.Available, false)
	wg.Wait()
}

func TestRunWorklaodManager(t *testing.T) {
	manager := newMockManager(t)

	watchers := interutils.NewWatchers()
	wch, err := watchers.Get()
	assert.Nil(t, err)
	go watchers.Run(context.Background())
	defer watchers.Stop()

	store := manager.store.(*storemocks.MockStore)
	svc := manager.svc.(*mocks.Service)
	initSVC(svc)
	// svc.On("VirtContext", mock.Anything).Return(nil)
	svc.On("WatchGuestEvents", mock.Anything).Return(wch, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	manager.startWorkloadManager(ctx)

	time.Sleep(2 * time.Second)

	assertInitStatus(t, store)
}
