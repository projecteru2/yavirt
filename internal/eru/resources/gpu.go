package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "embed"

	"github.com/kr/pretty"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/patrickmn/go-cache"

	"github.com/cockroachdb/errors"
	"github.com/jaypipes/ghw"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/vmcache"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

var execCommand = exec.Command

var (
	//go:embed gpu_name.json
	gpuNameJSON string
	gpuNameMap  = map[string]string{}
)

type SingleTypeGPUs struct {
	gpuMap  map[string]types.GPUInfo
	addrSet mapset.Set[string]
}

type GPUManager struct {
	mu            sync.Mutex
	gpuTypeMap    map[string]*SingleTypeGPUs
	coreMgr       *CoreResourcesManager
	lostGPUCache  *cache.Cache
	passthroughOK atomic.Bool
}

func initGPUNameMap(gpuProdMapCfg map[string]string) error {
	if len(gpuNameMap) > 0 {
		return nil
	}
	if err := json.Unmarshal([]byte(gpuNameJSON), &gpuNameMap); err != nil {
		return err
	}
	// merge name map from config to gpuNameMap
	for k, v := range gpuProdMapCfg {
		gpuNameMap[k] = v
	}
	return nil
}

func NewGPUManager(ctx context.Context, cfg *configs.Config, coreMgr *CoreResourcesManager) (*GPUManager, error) {
	if err := initGPUNameMap(cfg.Resource.GPUProductMap); err != nil {
		return nil, err
	}
	passthroughOK := checkPassthrough()
	var (
		gpuMap = make(map[string]*SingleTypeGPUs)
		err    error
	)
	if passthroughOK {
		if gpuMap, err = fetchGPUInfoFromHardware(); err != nil {
			return nil, err
		}
	}
	log.WithFunc("NewGPUManager").Infof(ctx, "hardware gpu info: %# v", pretty.Formatter(gpuMap))

	mgr := &GPUManager{
		gpuTypeMap:   gpuMap,
		coreMgr:      coreMgr,
		lostGPUCache: cache.New(3*time.Minute, 1*time.Minute),
	}
	mgr.passthroughOK.Store(passthroughOK)

	go mgr.monitor(ctx)
	return mgr, nil
}

func (g *GPUManager) GetResource() *gputypes.NodeResource {
	g.mu.Lock()
	defer g.mu.Unlock()

	res := &gputypes.NodeResource{
		ProdCountMap: make(gputypes.ProdCountMap),
	}
	for prod, gpus := range g.gpuTypeMap {
		res.ProdCountMap[prod] = len(gpus.gpuMap)
	}
	return res
}

func (g *GPUManager) alloc(req *gputypes.EngineParams, usedAddrsMap mapset.Set[string]) (ans []types.GPUInfo, err error) {
	totalCount := 0
	for reqProd, reqCount := range req.ProdCountMap {
		if reqCount <= 0 {
			continue
		}
		totalCount += reqCount
		singleTypeGPUs := g.gpuTypeMap[reqProd]
		available := singleTypeGPUs.addrSet.Difference(usedAddrsMap)
		if available.Cardinality() < reqCount {
			return nil, errors.New("no enough GPU")
		}
		for addr := range available.Iter() {
			if reqCount <= 0 {
				break
			}
			info := singleTypeGPUs.gpuMap[addr]
			ans = append(ans, info)
			reqCount--
		}
	}
	return ans, nil
}

func (g *GPUManager) Alloc(req *gputypes.EngineParams) (ans []types.GPUInfo, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	usedAddrs := vmcache.FetchGPUAddrs()
	usedAddrsSet := mapset.NewSet[string](usedAddrs...)
	return g.alloc(req, usedAddrsSet)
}

func (g *GPUManager) monitor(ctx context.Context) {
	logger := log.WithFunc("GPUManager.monitor")
	if !g.passthroughOK.Load() {
		logger.Warn(ctx, "passthrough is not setup yet, so monitor does nothing")
		// GPU passthrough is not setup yet, so we set gpu capacity to 0 here.
		g.coreMgr.UpdateGPU(g.GetResource())
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
		}
		gpuMap, err := fetchGPUInfoFromHardware()
		if err != nil {
			logger.Errorf(ctx, err, "failed to fetch gpu info from hardware")
			continue
		}

		totalGPUAddrSet := mapset.NewSet[string]()
		for _, gpus := range gpuMap {
			totalGPUAddrSet = totalGPUAddrSet.Union(gpus.addrSet)
		}
		// check if used gpu are lost
		usedAddrs := vmcache.FetchDomainGPUAddrs()

		for domain, addrs := range usedAddrs {
			addrsSet := mapset.NewSet[string](addrs...)
			diff := addrsSet.Difference(totalGPUAddrSet)
			if diff.Cardinality() > 0 {
				de := vmcache.FetchDomainEntry(domain)
				if de == nil {
					continue
				}
				lostGPUInfo := map[string]string{
					"node":      configs.Hostname(),
					"domain":    domain,
					"pci_addrs": strings.Join(diff.ToSlice(), ","),
					"app_id":    de.AppID,
					"app_sid":   de.AppSID,
					"appname":   de.AppName,
					"ip":        de.IP,
				}
				g.lostGPUCache.Set(domain, lostGPUInfo, cache.DefaultExpiration)
			}
		}

		g.mu.Lock()
		g.gpuTypeMap = gpuMap
		// don't use defer here,because the following GetRsource also need acquire locker
		g.mu.Unlock()

		g.coreMgr.UpdateGPU(g.GetResource())
	}
}

func checkPassthrough() bool {
	// err := execCommand("sh", "-c", "dmesg | grep -E 'DMAR|IOMMU'").Run()
	// return err == nil
	return true
}

func fetchGPUInfoFromHardware() (map[string]*SingleTypeGPUs, error) {
	pci, err := ghw.PCI()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pci info")
	}

	cmdOut, err := execCommand("lshw", "-quiet", "-json", "-C", "display").Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to run lshw")
	}
	params := []map[string]any{}
	if err = json.Unmarshal(cmdOut, &params); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal lshw output")
	}
	// map format:
	//
	// product:
	//   pciaddr: gpu info
	gpuMap := make(map[string]*SingleTypeGPUs)
	excludePCIs := map[string]struct{}{}
	for _, addr := range configs.Conf.Resource.ExcludePCIs {
		excludePCIs[addr] = struct{}{}
	}
	for _, param := range params {
		var addr string
		if businfoRaw, ok := param["businfo"]; ok && businfoRaw != nil {
			addr = strings.Split(businfoRaw.(string), "@")[1]
		} else if handleRaw, ok := param["handle"]; ok && handleRaw != nil {
			handle := handleRaw.(string) //nolint
			idx := strings.Index(handle, ":")
			addr = handle[idx+1:]
		}
		if addr == "" {
			log.Warnf(context.TODO(), "Can't fetch PCI address from %v", param)
			continue
		}
		if _, ok := excludePCIs[addr]; ok {
			log.Warnf(context.TODO(), "Exclude PCI address %s", addr)
			continue
		}
		deviceInfo := pci.GetDevice(addr)
		var numa string
		if deviceInfo != nil && deviceInfo.Node != nil {
			numa = fmt.Sprintf("%d", deviceInfo.Node.ID)
		}
		info := types.GPUInfo{
			Address: addr,
			Product: param["product"].(string),
			Vendor:  param["vendor"].(string),
			NumaID:  numa,
		}
		if strings.Contains(info.Vendor, "NVIDIA") || strings.Contains(info.Vendor, "AMD") {
			prod := gpuNameMap[info.Product]
			if prod == "" {
				return nil, fmt.Errorf("unknown GPU product: %s", info.Product)
			}
			singleTypeGPUs := gpuMap[prod]
			if singleTypeGPUs == nil {
				singleTypeGPUs = &SingleTypeGPUs{
					gpuMap:  make(map[string]types.GPUInfo),
					addrSet: mapset.NewSet[string](),
				}
			}
			singleTypeGPUs.gpuMap[info.Address] = info
			singleTypeGPUs.addrSet.Add(info.Address)
			gpuMap[prod] = singleTypeGPUs
		}
	}
	return gpuMap, nil
}
