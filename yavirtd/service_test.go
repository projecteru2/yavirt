package yavirtd

import (
	"context"
	"testing"

	"github.com/projecteru2/yavirt/model"
	"github.com/projecteru2/yavirt/test/assert"
	"github.com/projecteru2/yavirt/test/mock"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt"
	vg "github.com/projecteru2/yavirt/virt/guest"
	managerocks "github.com/projecteru2/yavirt/virt/guest/manager/mocks"
	virtypes "github.com/projecteru2/yavirt/virt/types"
)

func init() {
	model.Setup()
}

func TestCreateGuest(t *testing.T) {
	svc := testService(t)

	svc.guest.(*managerocks.Manageable).On("Create",
		mock.Anything, // ctx
		mock.Anything, // cpu
		mock.Anything, // memory
		mock.Anything, // vols
		mock.Anything, // imgName
		mock.Anything, // imgUser
		mock.Anything, // host
		mock.Anything, // dmiUUID
		mock.Anything, // labels
	).Return(testVirtGuest(t), nil)
	_, err := svc.CreateGuest(testVirtContext(t), virtypes.GuestCreateOption{
		CPU:       1,
		Mem:       util.GB,
		ImageName: "ubuntu",
		ImageUser: "anrs",
		Volumes:   nil,
		DmiUUID:   "uuid",
		Labels:    nil,
	})
	assert.NilErr(t, err)
}

func TestGetGuest(t *testing.T) {
	svc := testService(t)
	svc.guest.(*managerocks.Manageable).On("Load", mock.Anything, mock.Anything).Return(testVirtGuest(t), nil)
	_, err := svc.GetGuest(testVirtContext(t), "id")
	assert.NilErr(t, err)
}

func TestGetGuestUUID(t *testing.T) {
	svc := testService(t)
	svc.guest.(*managerocks.Manageable).On("LoadUUID", mock.Anything, mock.Anything).Return("uuid", nil)
	_, err := svc.GetGuestUUID(testVirtContext(t), "id")
	assert.NilErr(t, err)
}

func TestCopyToGuest(t *testing.T) {
	svc := testService(t)
	svc.guest.(*managerocks.Manageable).On("CopyToGuest",
		mock.Anything, // ctx
		mock.Anything, // id
		mock.Anything, // dest
		mock.Anything, // content
		mock.Anything, // override
	).Return(nil)
	err := svc.CopyToGuest(testVirtContext(t), "id", "dest", nil, true)
	assert.NilErr(t, err)
}

func testVirtGuest(t *testing.T) *vg.Guest {
	mg, err := model.NewGuest(nil, nil)
	assert.NilErr(t, err)
	assert.NotNil(t, mg)
	return vg.New(testVirtContext(t), mg)
}

func testVirtContext(t *testing.T) virt.Context {
	return virt.NewContext(context.Background(), nil)
}

func testService(t *testing.T) *Service {
	return &Service{
		Host:        &model.Host{},
		guest:       &managerocks.Manageable{},
		BootGuestCh: make(chan string, 1),
	}
}
