package recycle

import (
	"context"
	"strings"
	"testing"
	"time"

	virttypes "github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/service/mocks"
	"github.com/projecteru2/yavirt/pkg/notify/bison"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"

	coretypes "github.com/projecteru2/core/types"
	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/eru/types"
	grpcstatus "google.golang.org/grpc/status"
)

func TestDeleteGuest(t *testing.T) {
	bison.Setup(nil, t)
	deleteWait = 0
	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()
	err := Setup(ctx, nil, t)
	assert.Nil(t, err)

	svc := &mocks.Service{}
	mockSto := stor.(*storemocks.MockStore)

	// still in eru
	mockSto.On("GetWorkload", mock.Anything, mock.Anything).Return(
		&types.Workload{
			ID: virttypes.EruID("00033017009174384208170000000001"),
		}, nil).Once()
	err = deleteGuest(svc, virttypes.EruID("00033017009174384208170000000001"))
	assert.Err(t, err)
	assert.True(t, strings.Contains(err.Error(), "still in eru"))

	// get an other error from eru
	mockSto.On("GetWorkload", mock.Anything, mock.Anything).Return(nil, grpcstatus.Error(1111, coretypes.ErrInvaildCount.Error())).Once()
	err = deleteGuest(svc, virttypes.EruID("00033017009174384208170000000001"))
	assert.Err(t, err)

	mockSto.On("GetWorkload", mock.Anything, mock.Anything).Return(nil, grpcstatus.Error(1051, coretypes.ErrInvaildCount.Error())).Once()
	svc.On("VirtContext", mock.Anything).Return(nil).Once()
	svc.On("ControlGuest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	err = deleteGuest(svc, virttypes.EruID("00033017009174384208170000000001"))
	assert.Nil(t, err)

}

func TestNormal(t *testing.T) {
	interval = 2 * time.Second

	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()
	err := Setup(ctx, nil, t)
	assert.Nil(t, err)

	svc := &mocks.Service{}
	svc.On("GetGuestIDList", mock.Anything).Return([]string{"00033017009174384208170000000001", "00033017009174384208170000000002"}, nil)
	mockSto := stor.(*storemocks.MockStore)
	mockSto.On("ListNodeWorkloads", mock.Anything, mock.Anything).Return(
		[]*types.Workload{
			{
				ID: virttypes.EruID("00033017009174384208170000000001"),
			},
			{
				ID: virttypes.EruID("00033017009174384208170000000002"),
			},
		}, nil)
	Run(ctx, svc)
	time.Sleep(7 * time.Second)
}
