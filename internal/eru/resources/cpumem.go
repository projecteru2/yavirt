package resources

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/projecteru2/core/log"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	"github.com/projecteru2/yavirt/configs"
)

type CPUMemManager struct {
	cpumem  *cpumemtypes.NodeResource
	coreMgr *CoreResourcesManager
}

func NewCPUMemManager(coreMgr *CoreResourcesManager, cfg *configs.Config) (*CPUMemManager, error) {
	logger := log.WithFunc("NewCPUMemManager")
	cpumem, err := fetchCPUMemFromHardware(cfg)
	if err != nil {
		return nil, err
	}
	logger.Infof(context.TODO(), "hardware cpumem info: %+v", cpumem)
	go func() {
		// Core need to connect to the local grpc server, so sleep 30s here to wait local grpc server up
		time.Sleep(45 * time.Second)
		if err := coreMgr.UpdateCPUMem(cpumem); err != nil {
			logger.Errorf(context.TODO(), err, "failed to update cpumem")
		}
	}()
	return &CPUMemManager{
		cpumem:  cpumem,
		coreMgr: coreMgr,
	}, err
}

func fetchCPUMemFromHardware(cfg *configs.Config) (*cpumemtypes.NodeResource, error) {
	numa := cpumemtypes.NUMA{}
	numaMem := cpumemtypes.NUMAMemory{}

	cpu, err := ghw.CPU()
	if err != nil {
		return nil, err
	}
	mem, err := ghw.Memory()
	if err != nil {
		return nil, err
	}
	// 因为core会对memory取0.8，而在我们这个场景中，当机器的内存很大比如500G的时候，取0.8浪费太多
	// 所以这里默认只保留配置的ReservedMemory给yavirt，os使用。
	reservedMem := cfg.Resource.ReservedMemory
	if reservedMem > mem.TotalUsableBytes*20/100 {
		reservedMem = mem.TotalUsableBytes * 20 / 100
	}
	infoMem := (mem.TotalUsableBytes - reservedMem) * 100 / 80

	topology, err := ghw.Topology()
	if err != nil {
		return nil, err
	}
	numaReservedMem := reservedMem / int64(len(topology.Nodes))
	for _, node := range topology.Nodes {
		numaMem[strconv.Itoa(node.ID)] = (node.Memory.TotalUsableBytes - numaReservedMem) * 100 / 80
		for _, core := range node.Cores {
			for _, id := range core.LogicalProcessors {
				numa[strconv.Itoa(id)] = fmt.Sprintf("%d", node.ID)
			}
		}
	}
	return &cpumemtypes.NodeResource{
		CPU:        float64(cpu.TotalThreads),
		Memory:     infoMem,
		NUMAMemory: numaMem,
		NUMA:       numa,
	}, nil
}
