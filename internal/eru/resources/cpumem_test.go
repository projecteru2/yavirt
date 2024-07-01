package resources

import (
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/cpu"
	"github.com/jaypipes/ghw/pkg/memory"
	"github.com/jaypipes/ghw/pkg/topology"
	"github.com/mcuadros/go-defaults"
	"github.com/projecteru2/yavirt/configs"
	"github.com/stretchr/testify/assert"
)

func TestFetchCPUMem(t *testing.T) {
	patches := ApplyFuncReturn(ghw.Topology, &topology.Info{
		Nodes: []*topology.Node{
			{
				ID: 0,
				Cores: []*cpu.ProcessorCore{
					{
						ID: 0,
						LogicalProcessors: []int{
							0, 2,
						},
					},
				},
				Memory: &memory.Area{
					TotalUsableBytes: 1024,
				},
			},
			{
				ID: 1,
				Cores: []*cpu.ProcessorCore{
					{
						ID: 1,
						LogicalProcessors: []int{
							1, 3,
						},
					},
				},
				Memory: &memory.Area{
					TotalUsableBytes: 1024,
				},
			},
		},
	}, nil)
	defer patches.Reset()

	patches = ApplyFuncReturn(ghw.CPU, &cpu.Info{
		TotalThreads: 4,
	}, nil)
	defer patches.Reset()

	patches = ApplyFuncReturn(ghw.Memory, &memory.Info{
		Area: memory.Area{
			TotalUsableBytes: 2048,
		},
	}, nil)
	defer patches.Reset()
	cfg := &configs.Config{}
	defaults.SetDefaults(cfg)

	res, err := fetchCPUMemFromHardware(cfg)
	assert.Nil(t, err)
	assert.Equal(t, res.CPU, float64(4))
	assert.Equal(t, res.Memory, int64(2048))
	for core, node := range res.NUMA {
		switch node {
		case "0":
			assert.Truef(t, core == "0" || core == "2", "+++ %v", core)
		case "1":
			assert.Truef(t, core == "1" || core == "3", "++++ %v", core)
		default:
			assert.False(t, true)
		}
	}
	for _, node := range res.NUMAMemory {
		assert.Equal(t, node, int64(1025))
	}
}
