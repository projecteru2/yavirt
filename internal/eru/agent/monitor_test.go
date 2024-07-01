package agent

import (
	"context"
	"testing"
	"time"

	virttypes "github.com/projecteru2/libyavirt/types"
	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/service/mocks"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/stretchr/testify/assert"
)

func TestMonitor(t *testing.T) {
	manager := newMockManager(t)

	watchers := interutils.NewWatchers()
	wch, err := watchers.Get()
	assert.Nil(t, err)
	go watchers.Run(context.Background())
	defer watchers.Stop()

	store := manager.store.(*storemocks.MockStore)
	svc := manager.svc.(*mocks.Service)

	svc.On("VirtContext", mock.Anything).Return(nil)
	svc.On("WatchGuestEvents", mock.Anything).Return(wch, nil).Once()

	assert.Nil(t, store.GetMockWorkloadStatus("00033017009174384208170000000001"))
	assert.Nil(t, store.GetMockWorkloadStatus("00033017009174384208170000000002"))

	// stop "00033017009174384208170000000001"
	svc.On("GetGuest", mock.Anything, "00033017009174384208170000000001").Return(&virttypes.Guest{
		Resource: virttypes.Resource{
			ID: "00033017009174384208170000000001",
		},
		Labels:  map[string]string{"ERU": "1"},
		Running: false,
	}, nil)

	// svc.On("ExecuteGuest", mock.Anything, mock.Anything, mock.Anything).Return(&virttypes.ExecuteGuestMessage{}, nil)
	watchers.Watched(intertypes.Event{
		ID: "00033017009174384208170000000001",
		Op: "die",
	})

	// start monitor and wait for a while, then exit
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	go manager.initMonitor(ctx)
	time.Sleep(2 * time.Second)
	cancel()

	assert.Equal(t, store.GetMockWorkloadStatus("00033017009174384208170000000001"), &types.WorkloadStatus{
		ID:        "00033017009174384208170000000001",
		Nodename:  "fake",
		Extension: []byte(`{"ERU":"1"}`),
		Running:   false,
		Healthy:   false,
	})

	// start "00033017009174384208170000000002"
	wch2, err := watchers.Get()
	assert.Nil(t, err)
	svc.On("WatchGuestEvents", mock.Anything).Return(wch2, nil).Once()

	svc.On("GetGuest", mock.Anything, "00033017009174384208170000000002").Return(&virttypes.Guest{
		Resource: virttypes.Resource{
			ID: "00033017009174384208170000000002",
		},
		Labels:  map[string]string{"ERU": "1"},
		Running: true,
	}, nil)
	watchers.Watched(intertypes.Event{
		ID: "00033017009174384208170000000002",
		Op: "start",
	})

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	go manager.initMonitor(ctx)
	time.Sleep(2 * time.Second)
	cancel()

	assert.Equal(t, store.GetMockWorkloadStatus("00033017009174384208170000000001"), &types.WorkloadStatus{
		ID:        "00033017009174384208170000000001",
		Nodename:  "fake",
		Extension: []byte(`{"ERU":"1"}`),
		Running:   false,
		Healthy:   false,
	})
	time.Sleep(1 * time.Second)

	// initMonotr already stopped watcher and send an empty event,
	// so watchers got an chance to delete watcher
	watchers.Watched(intertypes.Event{})
	time.Sleep(1 * time.Second)
	assert.Zero(t, watchers.Len())
}
