package agent

import (
	"context"
	"testing"
	"time"

	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/service/mocks"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/stretchr/testify/assert"

	virttypes "github.com/projecteru2/libyavirt/types"
)

func assertInitStatus(t *testing.T, store *storemocks.MockStore) {
	assert.Equal(t, store.GetMockWorkloadStatus("00033017009174384208170000000001"), &types.WorkloadStatus{
		ID:        "00033017009174384208170000000001",
		Nodename:  "fake",
		Extension: []byte(`{"ERU":"1"}`),
		Running:   true,
		Healthy:   true,
	})

	assert.Equal(t, store.GetMockWorkloadStatus("00033017009174384208170000000002"), &types.WorkloadStatus{
		ID:        "00033017009174384208170000000002",
		Nodename:  "fake",
		Extension: []byte(`{"ERU":"1"}`),
		Running:   false,
		Healthy:   false,
	})
}

func initSVC(svc *mocks.Service) {
	svc.On("VirtContext", mock.Anything).Return(nil)
	svc.On("GetGuestIDList", mock.Anything).Return([]string{"00033017009174384208170000000001", "00033017009174384208170000000002"}, nil)
	svc.On("GetGuest", mock.Anything, "00033017009174384208170000000001").Return(&virttypes.Guest{
		Resource: virttypes.Resource{
			ID: "00033017009174384208170000000001",
		},
		Labels:  map[string]string{"ERU": "1"},
		Running: true,
	}, nil)
	svc.On("GetGuest", mock.Anything, "00033017009174384208170000000002").Return(&virttypes.Guest{
		Resource: virttypes.Resource{
			ID: "00033017009174384208170000000002",
		},
		Labels:  map[string]string{"ERU": "1"},
		Running: false,
	}, nil)
}
func TestHealthCheck(t *testing.T) {
	manager := newMockManager(t)
	svc := manager.svc.(*mocks.Service)
	initSVC(svc)

	ctx := context.Background()
	manager.checkAllWorkloads(ctx)
	store := manager.store.(*storemocks.MockStore)
	time.Sleep(2 * time.Second)

	assertInitStatus(t, store)
}
