package guest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/network"
	networkFactory "github.com/projecteru2/yavirt/internal/network/factory"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/guest/mocks"
	"github.com/projecteru2/yavirt/internal/volume"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/internal/volume/local"
	volmocks "github.com/projecteru2/yavirt/internal/volume/mocks"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	storemocks "github.com/projecteru2/yavirt/pkg/store/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
	utilmocks "github.com/projecteru2/yavirt/pkg/utils/mocks"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

const (
	MAX_RETRIES = 3
	INTERVAL    = 200 * time.Millisecond
)

func init() {
	idgen.Setup(0)
}

func newTestImage(username, name, tag, baseDir string) *vmitypes.Image {
	fullname := fmt.Sprintf("%s/%s:%s", username, name, tag)
	if username == "" {
		fullname = fmt.Sprintf("%s:%s", name, tag)
	}
	img, _ := vmiFact.NewImage(fullname)
	return img
}

func TestCreate_WithExtVolumes(t *testing.T) {
	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var genvol = func(id int64, cap int64) volume.Volume {
		vol, err := local.NewDataVolume("/data", cap)
		assert.NilErr(t, err)
		return vol
	}
	var extVols = []volume.Volume{genvol(1, utils.GB*10), genvol(2, utils.GB*20)}
	guest.AppendVols(extVols...)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusPending))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)

	bot.On("Define", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.DefineGuestForCreate(context.Background()))
	assert.Equal(t, meta.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusCreating))
}

func TestLifecycle(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	assert.Equal(t, meta.StatusPending, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusPending))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	bot.On("Define", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.DefineGuestForCreate(context.Background()))
	assert.Equal(t, meta.StatusCreating, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusCreating))

	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start(ctx, false))
	assert.Equal(t, meta.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusRunning))

	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(ctx, true))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}
	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, []volume.Volume{}))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	mgr := vmiFact.GetMockManager()
	mgr.On("Push", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Once()
	defer mgr.AssertExpectations(t)
	img := &vmitypes.Image{}

	bot.On("Capture", mock.Anything, mock.Anything).Return(img, nil).Once()
	bot.On("Close").Return(nil).Once()
	_, err := guest.Capture("anrs/aa", true)
	assert.NilErr(t, err)
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.Start(ctx, false))
	assert.Equal(t, meta.StatusRunning, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusRunning))

	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.Stop(ctx, true))
	assert.Equal(t, meta.StatusStopped, guest.Status)
	guest.rangeVolumes(checkVolsStatus(t, meta.StatusStopped))

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	// destroy always calls stop
	bot.On("Close").Return(nil).Once()
	bot.On("Shutdown", ctx, false).Return(nil).Once()

	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	done, err := guest.Destroy(ctx, false)
	assert.NilErr(t, err)
	assert.NilErr(t, <-done)
}

func checkVolsStatus(t *testing.T, expSt string) func(int, volume.Volume) bool {
	return func(_ int, v volume.Volume) bool {
		assert.Equal(t, expSt, v.GetStatus())
		return true
	}
}

func TestLifecycle_InvalidStatus(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	guest.Status = meta.StatusDestroyed
	assert.Err(t, guest.DefineGuestForCreate(context.Background()))
	assert.Err(t, guest.Stop(ctx, false))
	assert.Err(t, guest.Start(ctx, false))

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)

	_, err := guest.Capture("anrs/aa", true)
	assert.Err(t, err)

	guest.Status = meta.StatusResizing
	done, err := guest.Destroy(ctx, false)
	assert.Err(t, err)
	assert.Nil(t, done)

	guest.Status = meta.StatusPending

	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}

	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.Err(t, guest.Resize(cpumem, gpu, []volume.Volume{}))
}

func TestSyncState(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sto.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	guest.Status = meta.StatusCreating
	bot.On("Define", mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusDestroying
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusDestroying, true)
		return true
	})

	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusStopping
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusStopping, true)
		return true
	})
	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))

	guest.Status = meta.StatusStarting
	guest.rangeVolumes(func(_ int, v volume.Volume) bool {
		v.SetStatus(meta.StatusStarting, true)
		return true
	})
	bot.On("Boot", ctx).Return(nil).Once()
	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainShutoff, nil).Once()
	assert.NilErr(t, guest.SyncState(ctx))
}

func TestForceDestroy(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	sto, stoCancel := storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	sto.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	bot.On("Shutdown", ctx, true).Return(nil).Once()
	bot.On("Undefine").Return(nil).Once()
	bot.On("Close").Return(nil)
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	done, err := guest.Destroy(ctx, true)
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
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var guest, bot = newMockedGuest(t)
	defer bot.AssertExpectations(t)

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	bot.On("Close").Return(nil).Once()
	bot.On("GetState").Return(libvirt.DomainRunning, nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()

	guest.Status = meta.StatusRunning
	for _, vol := range guest.Vols {
		vol.SetStatus(meta.StatusRunning, true)
	}
	assert.NilErr(t, guest.SyncState(ctx))
}

func TestAmplifyOrigVols_HostDirMount(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	volmod, err := local.NewDataVolume("/tmp:/data", utils.GB)
	assert.NilErr(t, err)

	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil)
	bot.On("Unlock").Return()
	bot.On("AmplifyVolume", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Vols = volFact.Volumes{volmod}
	vol, err := local.NewDataVolume("/tmp:/data", utils.GB*10)
	assert.Nil(t, err)
	assert.NilErr(t, guest.amplifyOrigVol(vol, 20*utils.GB))
}

func TestAttachVolumes_CheckVolumeModel(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything).Return(nil, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	guest.HostName = "lo"
	guest.ID = "guestid"
	vol, err := local.NewDataVolume("/data", utils.GB)
	assert.Nil(t, err)
	vols := []volume.Volume{vol}
	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}

	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, vols))

	volmod := guest.Vols[1] // guest.Vols[0] is the sys volume.
	assert.True(t, len(volmod.GetID()) > 0)
	assert.Equal(t, guest.Status, volmod.GetStatus())
	assert.Equal(t, "/data", volmod.GetMountDir())
	// assert.Equal(t, "", volmod.HostDir)
	assert.Equal(t, utils.GB, volmod.GetSize())
	assert.Equal(t, guest.HostName, volmod.GetHostname())
	assert.Equal(t, guest.ID, volmod.GetGuestID())
}

func TestAttachVolumes_Rollback(t *testing.T) {
	var rolled bool
	rollback := func() { rolled = true }

	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachVolume", mock.Anything).Return(rollback, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("faked-error")).Once()

	guest.Status = meta.StatusRunning
	vol, err := local.NewDataVolume("/data", utils.GB)
	assert.Nil(t, err)
	vols := []volume.Volume{vol}

	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}

	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.Err(t, guest.Resize(cpumem, gpu, vols))
	assert.Equal(t, 1, guest.Vols.Len())
	// assert.Equal(t, models.VolSysType, guest.Vols[0].Type)
	assert.True(t, rolled)
}

// func TestCannotShrinkOrigVolumes(t *testing.T) {
// 	testcases := []struct {
// 		exists   string
// 		resizing string
// 	}{
// 		{"/data", "/data"},
// 		{"/data", "/tmp2:/data"},
// 		{"/tmp:/data", "/data"},
// 		{"/tmp:/data", "/tmp2:/data"},
// 	}

// 	for _, tc := range testcases {
// 		guest, _ := newMockedGuest(t)
// 		volmod, err := local.NewDataVolume(tc.exists, utils.GB*2)
// 		assert.NilErr(t, err)
// 		assert.NilErr(t, guest.AppendVols(volmod))

// 		guest.Status = meta.StatusRunning
// 		vol, err := local.NewDataVolume(tc.resizing, utils.GB)
// 		assert.Nil(t, err)
// 		vols := []volume.Volume{vol}

// 		cpumem := &cpumemtypes.EngineParams{
// 			CPU:    float64(guest.CPU),
// 			Memory: guest.Memory,
// 		}
// 		assert.True(t, errors.Contain(
// 			guest.Resize(cpumem, vols),
// 			errors.ErrCannotShrinkVolume,
// 		))
// 	}
// }

func TestAttachGPUs(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachGPUs", mock.Anything).Return(nil, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	guest.HostName = "lo"
	guest.ID = "guestid"
	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}

	// add 2 GPUs
	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 2,
		},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 2, guest.GPUEngineParams.Count())
	assert.Equal(t, gpu, guest.GPUEngineParams)

	// don't change
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 2, guest.GPUEngineParams.Count())
	assert.Equal(t, gpu, guest.GPUEngineParams)
	// add 2 GPUs
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("AttachGPUs", mock.Anything).Return(nil, nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	gpu = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 4,
		},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 4, guest.GPUEngineParams.Count())
	// assert.Equal(t, gpu, guest.GPUEngineParams)
}

func TestDetachGPUs(t *testing.T) {
	guest, bot := newMockedGuest(t)
	guest.GPUEngineParams = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 5,
		},
	}
	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("DetachGPUs", mock.Anything).Return(nil, nil).Once()

	sto, cancel := storemocks.Mock()
	defer sto.AssertExpectations(t)
	defer cancel()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	guest.Status = meta.StatusRunning
	guest.HostName = "lo"
	guest.ID = "guestid"
	cpumem := &cpumemtypes.EngineParams{
		CPU:    float64(guest.CPU),
		Memory: guest.Memory,
	}

	// detach 3 GPUs
	gpu := &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 2,
		},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 2, guest.GPUEngineParams.Count())
	assert.Equal(t, gpu, guest.GPUEngineParams)

	// don't change
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 2, guest.GPUEngineParams.Count())
	assert.Equal(t, gpu, guest.GPUEngineParams)
	// detach 2 GPUs
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("DetachGPUs", mock.Anything).Return(nil, nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	gpu = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 0,
		},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 0, guest.GPUEngineParams.Count())
	// assert.Equal(t, gpu, guest.GPUEngineParams)

	guest.GPUEngineParams = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{
			"nvidia-3070": 5,
		},
	}
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()
	bot.On("DetachGPUs", mock.Anything).Return(nil, nil).Once()
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	gpu = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.NilErr(t, guest.Resize(cpumem, gpu, nil))
	assert.Equal(t, 0, guest.GPUEngineParams.Count())
}

func TestInitSysDisk(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	var sto, stoCancel = storemocks.Mock()
	defer stoCancel()
	defer sto.AssertExpectations(t)
	sto.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	guest, bot := newMockedGuest(t)
	ciCfg := &types.CloudInitConfig{}
	gomonkey.ApplyMethodReturn(guest, "GenCloudInit", ciCfg, nil)
	gomonkey.ApplyMethodReturn(ciCfg, "GenerateISO", nil)
	gomonkey.ApplyMethodReturn(ciCfg, "ReplaceUserData", nil)

	mockVol := &volmocks.Volume{}
	mockVol.On("Cleanup").Return(nil).Once()
	mockVol.On("Delete", true).Return(nil).Once()
	mockVol.On("GetDevice").Return("vda").Once()
	mockVol.On("SetStatus", mock.Anything, mock.Anything).Return(nil).Once()
	guest.Vols[0] = mockVol

	newSysVol := &volmocks.Volume{}
	newSysVol.On("SetDevice", "vda").Return(nil).Once()
	newSysVol.On("SetGuestID", mock.Anything).Return(nil).Once()
	newSysVol.On("SetHostname", mock.Anything).Return(nil).Once()
	newSysVol.On("GenerateID").Return(nil).Once()
	newSysVol.On("SetStatus", mock.Anything, mock.Anything).Return(nil).Once()
	newSysVol.On("GetID").Return("xxxx").Once()
	newSysVol.On("Save").Return(nil).Once()
	newSysVol.On("PrepareSysDisk", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()

	bot.On("Shutdown", ctx, mock.Anything).Return(nil).Once()
	bot.On("ReplaceSysVolume", mock.Anything).Return(nil).Once()

	err := guest.InitSysDisk(ctx, guest.Img, &types.InitSysDiskArgs{
		Username: "test",
		Password: "test",
	}, newSysVol)
	assert.Nil(t, err)
	assert.Equal(t, meta.StatusStopped, guest.Status)
}

func TestFSFreeze(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	guest, bot := newMockedGuest(t)

	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()

	bot.On("FSFreezeAll", ctx).Return(2, nil).Once()

	nFS, err := guest.FSFreezeAll(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, nFS)
}

func TestFSThawAll(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	guest, bot := newMockedGuest(t)

	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()

	bot.On("FSThawAll", ctx).Return(2, nil).Once()

	nFS, err := guest.FSThawAll(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 2, nFS)
}

func TestFSFreezeStatus(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFn()

	guest, bot := newMockedGuest(t)

	defer bot.AssertExpectations(t)
	bot.On("Close").Return(nil).Once()
	bot.On("Trylock").Return(nil).Once()
	bot.On("Unlock").Return().Once()

	bot.On("FSFreezeStatus", ctx).Return("freezed", nil).Once()

	status, err := guest.FSFreezeStatus(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "freezed", status)
}

func newMockedGuest(t *testing.T) (*Guest, *mocks.Bot) {
	err := vmiFact.Setup(&vmitypes.Config{
		Type: "mock",
	})
	assert.Nil(t, err)
	networkFactory.Setup(&configs.NetworkConfig{})
	var bot = &mocks.Bot{}

	img, err := vmiFact.NewImage("user1/image1")
	img.VirtualSize = 1024
	assert.Nil(t, err)
	assert.Equal(t, img.Fullname(), "user1/image1:latest")
	gmod, err := models.NewGuest(models.NewHost(), img)
	gmod.JSONLabels = make(map[string]string)
	gmod.NetworkMode = network.FakeMode
	gmod.GPUEngineParams = &gputypes.EngineParams{
		ProdCountMap: gputypes.ProdCountMap{},
	}
	assert.NilErr(t, err)
	assert.Equal(t, gmod.ImageName, "user1/image1:latest")

	var guest = &Guest{
		Guest:  gmod,
		newBot: func(g *Guest) (Bot, error) { return bot, nil },
	}
	guest.IPNets = meta.IPNets{
		&meta.IPNet{},
	}

	return guest, bot
}
