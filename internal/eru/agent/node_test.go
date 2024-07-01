package agent

import (
	"context"
	"testing"
	"time"

	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/service/mocks"
	"github.com/projecteru2/yavirt/pkg/test/mock"

	"github.com/stretchr/testify/assert"
)

func TestNodeStatusReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := newMockManager(t)
	store := manager.store.(*storemocks.MockStore)
	svc := manager.svc.(*mocks.Service)
	svc.On("IsHealthy", mock.Anything).Return(true)

	status, err := store.GetNodeStatus(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, status.Alive, false)

	manager.nodeStatusReport(ctx)
	status, err = store.GetNodeStatus(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, status.Alive, true)
}

func TestHeartbeat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := newMockManager(t)
	store := manager.store.(*storemocks.MockStore)
	svc := manager.svc.(*mocks.Service)
	svc.On("IsHealthy", mock.Anything).Return(true)

	status, err := store.GetNodeStatus(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, status.Alive, false)

	go manager.heartbeat(ctx)

	time.Sleep(time.Duration(manager.config.HeartbeatInterval+2) * time.Second)
	status, err = store.GetNodeStatus(ctx, "fake")
	assert.Nil(t, err)
	assert.Equal(t, status.Alive, true)
}
