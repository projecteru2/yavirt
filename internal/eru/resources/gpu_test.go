package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/samber/lo"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestGPUAlloc(t *testing.T) {
	addrSet1 := mapset.NewSet[string]("0.0.0.0", "0.0.0.1", "0.0.0.2", "0.0.0.3")
	addrSet2 := mapset.NewSet[string]("0.0.0.4", "0.0.0.5", "0.0.0.6", "0.0.0.7")
	mgr := &GPUManager{
		gpuTypeMap: map[string]*SingleTypeGPUs{
			"nvidia-3090": {
				gpuMap: map[string]types.GPUInfo{
					"0.0.0.0": {
						Address: "0.0.0.0",
					},
					"0.0.0.1": {
						Address: "0.0.0.1",
					},
					"0.0.0.2": {
						Address: "0.0.0.2",
					},
					"0.0.0.3": {
						Address: "0.0.0.3",
					},
				},
				addrSet: addrSet1,
			},
			"nvidia-3070": {
				gpuMap: map[string]types.GPUInfo{
					"0.0.0.4": {
						Address: "0.0.0.4",
					},
					"0.0.0.5": {
						Address: "0.0.0.5",
					},

					"0.0.0.6": {
						Address: "0.0.0.6",
					},

					"0.0.0.7": {
						Address: "0.0.0.7",
					},
				},
				addrSet: addrSet2,
			},
		},
	}
	// no used addrs
	req := &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 1,
			"nvidia-3070": 2,
		},
	}
	usedAddrsSet := mapset.NewSet[string]()
	ans, err := mgr.alloc(req, usedAddrsSet)
	assert.Nil(t, err)
	assert.Len(t, ans, 3)

	allocedAddrs := lo.Map(ans, func(info types.GPUInfo, _ int) string {
		return info.Address
	})
	addrSet := mapset.NewSet[string](allocedAddrs...)
	diff1 := addrSet.Difference(addrSet2)
	diff2 := addrSet.Difference(addrSet1)
	assert.Equal(t, diff1.Cardinality(), 1)
	assert.Equal(t, diff2.Cardinality(), 2)

	// have used addrs
	usedAddrsSet = mapset.NewSet[string]("0.0.0.0", "0.0.0.3", "0.0.0.4", "0.0.0.5")
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Nil(t, err)
	assert.Len(t, ans, 3)
	allocedAddrs = lo.Map(ans, func(info types.GPUInfo, _ int) string {
		return info.Address
	})
	addrSet = mapset.NewSet[string](allocedAddrs...)
	diff1 = addrSet.Difference(addrSet2)
	diff2 = addrSet.Difference(addrSet1)
	assert.Equal(t, diff1.Cardinality(), 1)
	assert.Equal(t, diff2.Cardinality(), 2)

	assert.False(t, diff1.Contains("0.0.0.0"))
	assert.False(t, diff1.Contains("0.0.0.3"))
	assert.False(t, diff2.Contains("0.0.0.4"))
	assert.False(t, diff2.Contains("0.0.0.5"))

	// no enough resource
	req = &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 1,
			"nvidia-3070": 3,
		},
	}
	usedAddrsSet = mapset.NewSet[string]("0.0.0.0", "0.0.0.3", "0.0.0.4", "0.0.0.5")
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Error(t, err)
	req = &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 2,
			"nvidia-3070": 3,
		},
	}
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Error(t, err)
	req = &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 3,
			"nvidia-3070": 3,
		},
	}
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Error(t, err)
	req = &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 3,
			"nvidia-3070": 2,
		},
	}
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Error(t, err)
	req = &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			"nvidia-3090": 2,
			"nvidia-3070": 2,
		},
	}
	ans, err = mgr.alloc(req, usedAddrsSet)
	assert.Nil(t, err)
}

func TestFetchGPU(t *testing.T) {
	err := initGPUNameMap(nil)
	assert.Nil(t, err)
	execCommand = fakeExecCommand
	defer func() { execCommand = exec.Command }()

	gpuMap, err := fetchGPUInfoFromHardware()
	assert.Nil(t, err)
	assert.Len(t, gpuMap, 3)
	gpus3090, ok := gpuMap["nvidia-3090"]
	assert.True(t, ok)
	assert.Equal(t, gpus3090.addrSet.Cardinality(), 1)
	assert.True(t, gpus3090.addrSet.Contains("0000:3d:00.0"))

	gpus3070, ok := gpuMap["nvidia-3070"]
	assert.True(t, ok)
	assert.Equal(t, gpus3070.addrSet.Cardinality(), 1)
	assert.True(t, gpus3070.addrSet.Contains("0000:3c:00.0"))
	gpuMapJSON, _ := json.Marshal(gpuMap)
	t.Logf("%s", string(gpuMapJSON))

	mi210, ok := gpuMap["amd-mi210"]
	assert.True(t, ok)
	assert.Equal(t, mi210.addrSet.Cardinality(), 1)
	assert.True(t, mi210.addrSet.Contains("0000:e3:00.0"))
}

var fakeExecResult = `[
  {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:3d:00.0",
    "description" : "VGA compatible controller",
    "product" : "GA102 [GeForce RTX 3090]",
    "vendor" : "NVIDIA Corporation",
    "physid" : "0",
    "businfo" : "pci@0000:3d:00.0",
    "version" : "a1",
    "width" : 64,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "vfio-pci",
      "latency" : "0"
    },
    "capabilities" : {
      "pm" : "Power Management",
      "msi" : "Message Signalled Interrupts",
      "pciexpress" : "PCI Express",
      "vga_controller" : true,
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  },
  {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:3c:00.0",
    "description" : "VGA compatible controller",
    "product" : "GA104 [GeForce RTX 3070]",
    "vendor" : "NVIDIA Corporation",
    "physid" : "0",
    "businfo" : "pci@0000:3c:00.0",
    "version" : "a1",
    "width" : 64,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "vfio-pci",
      "latency" : "0"
    },
    "capabilities" : {
      "pm" : "Power Management",
      "msi" : "Message Signalled Interrupts",
      "pciexpress" : "PCI Express",
      "vga_controller" : true,
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  },
  {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:e3:00.0",
    "description" : "Display controller",
    "product" : "Aldebaran",
    "vendor" : "Advanced Micro Devices, Inc. [AMD/ATI]",
    "physid" : "0",
    "businfo" : "pci@0000:e3:00.0",
    "version" : "02",
    "width" : 64,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "amdgpu",
      "latency" : "0"
    },
    "capabilities" : {
      "pm" : "Power Management",
      "pciexpress" : "PCI Express",
      "msi" : "Message Signalled Interrupts",
      "msix" : "MSI-X",
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  }
]`

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{
		"GO_WANT_HELPER_PROCESS=1",
		fmt.Sprintf("GOCOVERDIR=%s", os.TempDir()),
	}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprint(os.Stdout, fakeExecResult)
	os.Exit(0)
}
