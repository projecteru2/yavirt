package resources

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/common"
	"github.com/projecteru2/yavirt/internal/eru/store"
	corestore "github.com/projecteru2/yavirt/internal/eru/store/core"
	storemocks "github.com/projecteru2/yavirt/internal/eru/store/mocks"
	"github.com/projecteru2/yavirt/internal/types"
	bdtypes "github.com/yuyang0/resource-bandwidth/bandwidth/types"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

var (
	mgr *Manager
	cli store.Store
)

type Manager struct {
	cfg     *configs.Config
	coreMgr *CoreResourcesManager

	gpu     *GPUManager
	cpumem  *CPUMemManager
	sto     *StorageManager
	gpuLock sync.Mutex
}

func (mgr *Manager) AllocGPU(req *gputypes.EngineParams) (ans []types.GPUInfo, err error) {
	return mgr.gpu.Alloc(req)
}

func (mgr *Manager) LockGPU() {
	mgr.gpuLock.Lock()
}

func (mgr *Manager) UnlockGPU() {
	mgr.gpuLock.Unlock()
}

func (mgr *Manager) FetchResources() (map[string][]byte, error) {
	cpumemBytes, err := json.Marshal(mgr.cpumem.cpumem)
	if err != nil {
		return nil, err
	}
	stoBytes, err := json.Marshal(mgr.sto.sto)
	if err != nil {
		return nil, err
	}
	gpusBytes, err := json.Marshal(mgr.gpu.GetResource())
	if err != nil {
		return nil, err
	}
	bd := bdtypes.NodeResource{
		Bandwidth: mgr.cfg.Resource.Bandwidth,
	}
	bdBytes, _ := json.Marshal(bd)

	ans := map[string][]byte{
		"cpumem":    cpumemBytes,
		"storage":   stoBytes,
		"gpu":       gpusBytes,
		"bandwidth": bdBytes,
	}
	return ans, nil
}

func (mgr *Manager) FetchCPUMem() *cpumemtypes.NodeResource {
	return mgr.cpumem.cpumem
}

func (mgr *Manager) FetchStorage() *stotypes.NodeResource {
	return mgr.sto.sto
}

func (mgr *Manager) FetchGPU() *gputypes.NodeResource {
	return mgr.gpu.GetResource()
}

func GetManager() *Manager {
	return mgr
}

func Setup(ctx context.Context, cfg *configs.Config, t *testing.T) (*Manager, error) {
	if t == nil {
		corestore.Init(ctx, &cfg.Eru)
		if cli = corestore.Get(); cli == nil {
			return nil, common.ErrGetStoreFailed
		}
	} else {
		cli = storemocks.NewFakeStore()
	}
	coreMgr := NewCoreResourcesManager()
	gpuMgr, err := NewGPUManager(ctx, cfg, coreMgr)
	if err != nil {
		return nil, err
	}
	cpumemMgr, err := NewCPUMemManager(coreMgr, cfg)
	if err != nil {
		return nil, err
	}
	stoMgr, err := newStorageManager()
	if err != nil {
		return nil, err
	}
	mgr = &Manager{
		cfg:     cfg,
		gpu:     gpuMgr,
		cpumem:  cpumemMgr,
		sto:     stoMgr,
		coreMgr: coreMgr,
	}
	return mgr, nil
}
