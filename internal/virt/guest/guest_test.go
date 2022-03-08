package guest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/guest/mocks"
	"github.com/projecteru2/yavirt/internal/virt/volume"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	storemocks "github.com/projecteru2/yavirt/pkg/store/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
	utilmocks "github.com/projecteru2/yavirt/pkg/utils/mocks"
)

const (
	MAX_RETRIES = 3
	INTERVAL    = 200 * time.Millisecond
)

func init() {
	idgen.Setup(0, time.Now())
	models.Setup()
}

func TestCreate_WithExtVolumes(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var genvol = func(id int64, cap int64) *models.Volume {
		vol, err := models.NewDataVolume("/data", cap)
		assert.NilErr(t, err)
		return vol
	}
	var extVols = []*models.Volume{genvol(1, utils.GB*10), genvol(2, utils.GB*20)}
	guest.AppendVols(extVols...)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusPending))

	var meta, metaCancel = storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)

	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.Create())
	assert.Equal(t, models.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusCreating))
}

func TestLifecycle(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	assert.Equal(t, models.StatusPending, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusPending))

	var meta, metaCancel = storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Create())
	assert.Equal(t, models.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusCreating))

	bot.On("Boot").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start())
	assert.Equal(t, models.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusRunning))

	bot.On("Shutdown", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(true))
	assert.Equal(t, models.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusStopped))

	assert.NilErr(t, guest.Resize(guest.CPU, guest.Memory, map[string]int64{}))
	assert.Equal(t, models.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusStopped))

	bot.On("Capture", mock.Anything, mock.Anything).Return(models.NewUserImage("anrs", "aa", 1024), nil).Once()
	bot.On("Close").Return(nil).Once()
	meta.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	_, err := guest.Capture("anrs", "aa", true)
	assert.NilErr(t, err)
	assert.Equal(t, models.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusStopped))

	bot.On("Boot").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start())
	assert.Equal(t, models.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusRunning))

	bot.On("Shutdown", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(true))
	assert.Equal(t, models.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, models.StatusStopped))

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	meta.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	meta.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	done, err := guest.Destroy(false)
	assert.NilErr(t, err)
	assert.NilErr(t, <-done)
}

func checkVolsStatus(t *testing.T, expSt string) func(int, volume.Virt) bool {
	return func(_ int, v volume.Virt) bool {
		assert.Equal(t, expSt, v.Model().Status)
		return true
	}
}

func TestLifecycle_InvalidStatus(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	guest.Status = models.StatusDestroyed
	assert.Err(t, guest.Create())
	assert.Err(t, guest.Stop(false))
	assert.Err(t, guest.Start())

	var meta, metaCancel = storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)

	meta.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	_, err := guest.Capture("anrs", "aa", true)
	assert.Err(t, err)

	guest.Status = models.StatusResizing
	done, err := guest.Destroy(false)
	assert.Err(t, err)
	assert.Nil(t, done)

	guest.Status = models.StatusPending
	assert.Err(t, guest.Resize(guest.CPU, guest.Memory, map[string]int64{}))
}

func TestSyncState(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var meta, metaCancel = storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	guest.Status = models.StatusCreating
	bot.On("Create").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.SyncState())

	guest.Status = models.StatusDestroying
	guest.rangeVolumes(func(_ int, v volume.Virt) bool {
		mod := v.Model()
		mod.Status = models.StatusDestroying
		return true
	})

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	meta.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	meta.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	assert.NilErr(t, guest.SyncState())

	guest.Status = models.StatusStopping
	guest.rangeVolumes(func(_ int, v volume.Virt) bool {
		mod := v.Model()
		mod.Status = models.StatusStopping
		return true
	})
	bot.On("Shutdown", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.SyncState())

	guest.Status = models.StatusStarting
	guest.rangeVolumes(func(_ int, v volume.Virt) bool {
		mod := v.Model()
		mod.Status = models.StatusStarting
		return true
	})
	bot.On("Boot").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.SyncState())
}

func TestForceDestroy(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	mutex := mockMutex()
	defer mutex.AssertExpectations(t)

	meta, metaCancel := storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)
	meta.On("NewMutex", mock.Anything).Return(mutex, nil).Once()
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	meta.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = models.StatusRunning
	bot.On("Shutdown", true).Return(nil).Once()
	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	done, err := guest.Destroy(true)
	assert.NilErr(t, err)
	assert.NilErr(t, <-done)
}

func mockMutex() *utilmocks.Locker {
	var unlock utils.Unlocker = func(context.Context) error {
		return nil
	}

	mutex := &utilmocks.Locker{}
	mutex.On("Lock", mock.Anything).Return(unlock, nil)

	return mutex
}

func TestSyncStateSkipsRunning(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainRunning, nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	guest.Status = models.StatusRunning
	assert.NilErr(t, guest.SyncState())
}

func TestAmplifyOrigVols_HostDirMount(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	volmod, err := models.NewDataVolume("/tmp:/data", utils.GB)
	assert.NilErr(t, err)

	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	bot.On("AmplifyVolume", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Vols = models.Volumes{volmod}
	mnt := map[string]int64{"/tmp:/data": utils.GB * 10}
	assert.NilErr(t, guest.amplifyOrigVols(mnt))
}

func TestAttachVolumes_CheckVolumeModel(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything, mock.Anything).Return(nil, nil).Once()

	meta, cancel := storemocks.Mock()
	defer meta.AssertExpectations(t)
	defer cancel()
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = models.StatusRunning
	guest.HostName = "lo"
	guest.ID = "guestid"
	vols := map[string]int64{"/data": utils.GB}
	assert.NilErr(t, guest.Resize(guest.CPU, guest.Memory, vols))

	volmod := guest.Vols[1] // guest.Vols[0] is the sys volume.
	assert.True(t, len(volmod.ID) > 0)
	assert.Equal(t, guest.Status, volmod.Status)
	assert.Equal(t, models.VolDataType, volmod.Type)
	assert.Equal(t, "/data", volmod.MountDir)
	assert.Equal(t, "", volmod.HostDir)
	assert.Equal(t, utils.GB, volmod.Capacity)
	assert.Equal(t, models.VolQcow2Format, volmod.Format)
	assert.Equal(t, guest.HostName, volmod.HostName)
	assert.Equal(t, guest.ID, volmod.GuestID)
}

func TestAttachVolumes_Rollback(t *testing.T) {
	var rolled bool
	rollback := func() { rolled = true }

	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything, mock.Anything).Return(rollback, nil).Once()

	meta, cancel := storemocks.Mock()
	defer meta.AssertExpectations(t)
	defer cancel()
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("faked-error")).Once()

	guest.Status = models.StatusRunning
	vols := map[string]int64{"/data": utils.GB}
	assert.Err(t, guest.Resize(guest.CPU, guest.Memory, vols))
	assert.Equal(t, 1, guest.Vols.Len())
	assert.Equal(t, models.VolSysType, guest.Vols[0].Type)
	assert.True(t, rolled)
}

func TestCannotShrinkOrigVolumes(t *testing.T) {
	testcases := []struct {
		exists   string
		resizing string
	}{
		{"/data", "/data"},
		{"/data", "/tmp2:/data"},
		{"/tmp:/data", "/data"},
		{"/tmp:/data", "/tmp2:/data"},
	}

	for _, tc := range testcases {
		guest, _ := newMockedGuest(t)
		volmod, err := models.NewDataVolume(tc.exists, utils.GB*2)
		assert.NilErr(t, err)
		assert.NilErr(t, guest.AppendVols(volmod))

		guest.Status = models.StatusRunning
		vols := map[string]int64{tc.resizing: utils.GB}
		assert.True(t, errors.Contain(
			guest.Resize(guest.CPU, guest.Memory, vols),
			errors.ErrCannotShrinkVolume,
		))
	}
}

func newMockedGuest(t *testing.T) (*Guest, *mocks.Bot) {
	var bot = &mocks.Bot{}

	gmod, err := models.NewGuest(models.NewHost(), models.NewSysImage())
	assert.NilErr(t, err)

	var guest = &Guest{
		Guest:  gmod,
		newBot: func(g *Guest) (Bot, error) { return bot, nil },
	}

	return guest, bot
}
