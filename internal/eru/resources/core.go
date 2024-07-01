package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/projecteru2/core/log"
	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	resourcetypes "github.com/projecteru2/core/resource/types"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/types"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/pkg/notify/bison"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

// CoreResourcesManager used to cache the resources on core
type CoreResourcesManager struct {
	mu     sync.Mutex
	cpumem *cpumemtypes.NodeResource
	gpu    *gputypes.NodeResource
	sto    *stotypes.NodeResource
}

func NewCoreResourcesManager() *CoreResourcesManager {
	return &CoreResourcesManager{}
}

func (cm *CoreResourcesManager) fetchResourcesWithLock() {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	logger := log.WithFunc("fetchResources")

	resp, err := cli.GetNodeResource(ctx, configs.Hostname())
	if err != nil {
		logger.Errorf(ctx, err, "failed to fetch resource from core")
		return
	}
	capacity := resp.Capacity
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for resName, rawParams := range capacity {
		switch resName {
		case intertypes.PluginNameCPUMem:
			cpumem := cpumemtypes.NodeResource{}
			if err = mapstructure.Decode(rawParams, &cpumem); err != nil {
				logger.Errorf(ctx, err, "failed to unmarshal resource cpumem")
			} else {
				logger.Debugf(ctx, "[core fetchResources] cpumem: %v", cpumem)
				cm.cpumem = &cpumem
			}
		case intertypes.PluginNameStorage:
			sto := stotypes.NodeResource{}
			if err = mapstructure.Decode(rawParams, &sto); err != nil {
				logger.Errorf(ctx, err, "failed to unmarshal resource storage")
			} else {
				logger.Debugf(ctx, "[core fetchResources] storage: %v", sto)
				cm.sto = &sto
			}
		case intertypes.PluginNameGPU:
			gpu := gputypes.NodeResource{}
			if err = mapstructure.Decode(rawParams, &gpu); err != nil {
				logger.Errorf(ctx, err, "failed to unmarshal resource gpu")
			} else {
				logger.Debugf(ctx, "[core fetchResources] gpu: %v", gpu)
				cm.gpu = &gpu
			}
		}
	}
}

func (cm *CoreResourcesManager) GetCpumem() (ans *cpumemtypes.NodeResource) {
	cm.mu.Lock()
	ans = cm.cpumem
	cm.mu.Unlock()
	if ans != nil {
		return
	}
	cm.fetchResourcesWithLock()
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.cpumem
}

func (cm *CoreResourcesManager) GetGPU() (ans *gputypes.NodeResource) {
	cm.mu.Lock()
	ans = cm.gpu
	cm.mu.Unlock()
	if ans != nil {
		return
	}
	cm.fetchResourcesWithLock()
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.gpu
}

func (cm *CoreResourcesManager) UpdateGPU(nr *gputypes.NodeResource) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	logger := log.WithFunc("UpdateGPU")

	remoteNR := cm.GetGPU()
	if remoteNR == nil {
		return
	}
	if remoteNR.Count() == nr.Count() {
		remoteNR1 := remoteNR.DeepCopy()
		remoteNR1.Sub(nr)
		if remoteNR1.Count() == 0 {
			logger.Debug(ctx, "remote gpu config is consistent")
			return
		}
	}
	logger.Infof(ctx, "start to update gpu resource: <local:%v, remote:%v>", nr, remoteNR)
	resBS, _ := json.Marshal(nr)
	opts := &types.SetNodeOpts{
		Nodename: configs.Hostname(),
		Delta:    false,
		Resources: map[string][]byte{
			"gpu": resBS,
		},
	}
	_, err := cli.SetNode(ctx, opts)
	if err != nil {
		logger.Errorf(ctx, err, "failed to update core resource")
		return
	}

	notifier := bison.GetService()
	if notifier != nil {
		text := fmt.Sprintf(`
<font color=#00CC33 size=10>update core gpu resource successfully</font>

---

- **node:** %s
- **gpu:** %v
		`, configs.Hostname(), nr)
		_ = notifier.SendMarkdown(ctx, "update core gpu resource successfully", text)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.gpu = nr
}

func (cm *CoreResourcesManager) UpdateCPUMem(nr *cpumemtypes.NodeResource) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()
	logger := log.WithFunc("UpdateCPUMem")

	localNR := nr.DeepCopy()
	remoteNR := cm.GetCpumem()
	if remoteNR == nil {
		return err
	}
	if remoteNR.CPU == localNR.CPU && remoteNR.Memory <= localNR.Memory && remoteNR.Memory >= (localNR.Memory*75/100) {
		logger.Info(ctx, "remote cpumem config is consistent")
		return err
	}
	logger.Infof(ctx, "start to update cpumem resource: <local:%+v, remote:%+v>", localNR, remoteNR)
	// prepare data for SetNode
	cb, _ := convCpumemBytes(localNR)
	opts := &types.SetNodeOpts{
		Nodename: configs.Hostname(),
		Delta:    false,
		Resources: map[string][]byte{
			"cpumem": cb,
		},
	}
	if _, err = cli.SetNode(ctx, opts); err != nil {
		logger.Errorf(ctx, err, "failed to update core resource")
		return err
	}

	notifier := bison.GetService()
	if notifier != nil {
		text := fmt.Sprintf(`
<font color=#00CC33 size=10>update core cpumem resource successfully</font>
---

- **node:** %s
- **cpumem:** %+v
		 `, configs.Hostname(), localNR)
		_ = notifier.SendMarkdown(ctx, "update core cpumem resource successfully", text)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.cpumem = localNR
	return nil
}

func convCpumemBytes(localNR *cpumemtypes.NodeResource) ([]byte, error) {
	cpumem := resourcetypes.RawParams{
		"cpu":    localNR.CPU,
		"memory": localNR.Memory * 80 / 100, // use 80% of memory
	}
	// nodeID => cpuID list
	numaCPUMap := map[string][]string{}
	for cpuID, numID := range localNR.NUMA {
		numaCPUMap[numID] = append(numaCPUMap[numID], cpuID)
	}
	numaCPUList := make([]string, 0, len(numaCPUMap))
	for idx := 0; idx < len(numaCPUMap); idx++ {
		cpuIDList := numaCPUMap[strconv.Itoa(idx)]
		// sort here, so we can write UT easily
		sort.Strings(cpuIDList)
		numaCPUList = append(numaCPUList, strings.Join(cpuIDList, ","))
	}
	if len(numaCPUList) > 0 {
		cpumem["numa-cpu"] = numaCPUList
	}
	numaMemList := make([]string, 0, len(localNR.NUMAMemory))
	for idx := 0; idx < len(localNR.NUMAMemory); idx++ {
		nodeMem := localNR.NUMAMemory[strconv.Itoa(idx)] * 80 / 100 // use 80% of memory
		numaMemList = append(numaMemList, strconv.FormatInt(nodeMem, 10))
	}
	if len(numaMemList) > 0 {
		cpumem["numa-memory"] = numaMemList
	}
	return json.Marshal(cpumem)
}
