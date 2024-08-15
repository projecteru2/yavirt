package boar

import (
	"encoding/json"

	"strings"

	pb "github.com/projecteru2/core/rpc/gen"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/hostdir"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/internal/volume/rbd"

	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
	hostdirtypes "github.com/yuyang0/resource-hostdir/hostdir/types"
	rbdtypes "github.com/yuyang0/resource-rbd/rbd/types"
)

func extractCPUMem(resources map[string][]byte) (eParams *cpumemtypes.EngineParams, err error) {
	cpumemRaw, ok := resources[intertypes.PluginNameCPUMem]
	if !ok {
		return nil, nil //nolint
	}
	var ans cpumemtypes.EngineParams
	err = json.Unmarshal(cpumemRaw, &ans)
	return &ans, err
}

func extractGPU(resources map[string][]byte) (eParams *gputypes.EngineParams, err error) {
	gpuRaw, ok := resources[intertypes.PluginNameGPU]
	if !ok {
		return nil, nil //nolint
	}
	var ans gputypes.EngineParams
	err = json.Unmarshal(gpuRaw, &ans)
	return &ans, err
}

func extractVols(resources map[string][]byte) ([]volume.Volume, error) { //nolint
	var sysVol volume.Volume
	vols := make([]volume.Volume, 1) // first place is for sys volume
	appendVol := func(vol volume.Volume) error {
		if vol.IsSys() {
			if sysVol != nil {
				return errors.New("multiple sys volume")
			}
			sysVol = vol
			return nil
		}
		vols = append(vols, vol) //nolint
		return nil
	}

	if stoResRaw, ok := resources[intertypes.PluginNameStorage]; ok {
		eParams := &stotypes.EngineParams{}
		if err := json.Unmarshal(stoResRaw, eParams); err != nil {
			return nil, errors.Wrap(err, "")
		}
		for _, part := range eParams.Volumes {
			vol, err := local.NewVolumeFromStr(part)
			if err != nil {
				return nil, err
			}
			if err := appendVol(vol); err != nil {
				return nil, err
			}
		}
	}
	if rbdResRaw, ok := resources[intertypes.PluginNameRBD]; ok {
		eParams := &rbdtypes.EngineParams{}
		if err := json.Unmarshal(rbdResRaw, eParams); err != nil {
			return nil, errors.Wrap(err, "")
		}
		for _, part := range eParams.Volumes {
			vol, err := rbd.NewFromStr(part)
			if err != nil {
				return nil, err
			}
			if err := appendVol(vol); err != nil {
				return nil, err
			}
		}
	}
	if hostdirResRaw, ok := resources[intertypes.PluginNameHostdir]; ok {
		eParams := &hostdirtypes.EngineParams{}
		if err := json.Unmarshal(hostdirResRaw, eParams); err != nil {
			return nil, errors.Wrap(err, "")
		}
		for _, part := range eParams.Volumes {
			vol, err := hostdir.NewFromStr(part)
			if err != nil {
				return nil, err
			}
			if err := appendVol(vol); err != nil {
				return nil, err
			}
		}
	}
	if sysVol != nil {
		vols[0] = sysVol
	} else {
		vols = vols[1:]
	}
	return vols, nil
}

func convGuestResp(g *models.Guest) (resp *types.Guest) {
	resp = &types.Guest{}
	resp.ID = g.ID
	resp.Hostname = g.HostName
	resp.Status = g.Status
	resp.CreateTime = g.CreatedTime
	resp.UpdateTime = g.UpdatedTime
	resp.ImageName = g.ImageName
	resp.CPU = g.CPU
	resp.Mem = g.Memory
	resp.Labels = g.JSONLabels
	resp.Running = (g.Status == meta.StatusRunning)

	if len(g.IPs) > 0 {
		var ips = make([]string, len(g.IPs))
		for i, ip := range g.IPs {
			ips[i] = ip.IPAddr()
		}
		resp.IPs = ips
		resp.Networks = map[string]string{"IP": strings.Join(ips, ", ")}
	}

	return
}

// ConvSetWorkloadsStatusOptions .
func ConvSetWorkloadsStatusOptions(gss []types.EruGuestStatus) *pb.SetWorkloadsStatusOptions {
	css := make([]*pb.WorkloadStatus, len(gss))
	for i, gs := range gss {
		css[i] = convWorkloadStatus(gs)
	}

	return &pb.SetWorkloadsStatusOptions{
		Status: css,
	}
}

func convWorkloadStatus(gs types.EruGuestStatus) *pb.WorkloadStatus {
	return &pb.WorkloadStatus{
		Id:       gs.EruGuestID,
		Running:  gs.Running,
		Healthy:  gs.Healthy,
		Ttl:      int64(gs.TTL.Seconds()),
		Networks: map[string]string{"IP": gs.GetIPAddrs()},
	}
}
