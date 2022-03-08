package yavirtd

import (
	"strings"

	pb "github.com/projecteru2/core/rpc/gen"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/models"
)

func convGuestResp(g *model.Guest) (resp *types.Guest) {
	resp = &types.Guest{}
	resp.ID = types.EruID(g.ID)
	resp.Status = g.Status
	resp.CreateTime = g.CreatedTime
	resp.UpdateTime = g.UpdatedTime
	resp.ImageName = g.ImageName
	resp.ImageUser = g.ImageUser
	resp.CPU = g.CPU
	resp.Mem = g.Memory

	if len(g.IPs) > 0 {
		var ips = make([]string, len(g.IPs))
		for i, ip := range g.IPs {
			ips[i] = ip.IPAddr()
		}
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
