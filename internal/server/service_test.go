package server

import (
	"context"
	"testing"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt"
	vg "github.com/projecteru2/yavirt/internal/virt/guest"
	managerocks "github.com/projecteru2/yavirt/internal/virt/guest/manager/mocks"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func init() {
	models.Setup()
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
		Mem:       utils.GB,
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

func TestGetGuestIDList(t *testing.T) {
	localIDs := []string{"ya0", "ya1", "ya2"}
	svc := testService(t)
	svc.guest.(*managerocks.Manageable).On("ListLocalIDs", mock.Anything, mock.Anything).Return(localIDs, nil).Once()

	ids, err := svc.GetGuestIDList(testVirtContext(t))
	assert.NilErr(t, err)

	eruIDs := []string{types.EruID("ya0"), types.EruID("ya1"), types.EruID("ya2")}
	assert.Equal(t, eruIDs, ids)
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
	mg, err := models.NewGuest(nil, nil)
	assert.NilErr(t, err)
	assert.NotNil(t, mg)
	return vg.New(testVirtContext(t), mg)
}

func testVirtContext(t *testing.T) virt.Context {
	return virt.NewContext(context.Background(), nil)
}

func testService(t *testing.T) *Service {
	return &Service{
		Host:        &models.Host{},
		guest:       &managerocks.Manageable{},
		BootGuestCh: make(chan string, 1),
	}
}
