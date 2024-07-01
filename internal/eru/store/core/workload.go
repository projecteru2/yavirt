package core

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	pb "github.com/projecteru2/core/rpc/gen"
	virttypes "github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/utils"
)

func getCacheTTL(ttl int64) time.Duration {
	n, _ := rand.Int(rand.Reader, big.NewInt(ttl))
	delta := n.Int64() / 4
	ttl = ttl - ttl/8 + delta
	return time.Duration(ttl) * time.Second
}

// SetWorkloadStatus deploy containers
func (c *Store) SetWorkloadStatus(ctx context.Context, status *types.WorkloadStatus, ttl int64) error {
	workloadStatus := fmt.Sprintf("%+v", status)
	if ttl == 0 {
		cached, ok := c.cache.Get(status.ID)
		if ok {
			str := cached.(string) //nolint
			if str == workloadStatus {
				return nil
			}
		}
	}

	statusPb := &pb.WorkloadStatus{
		Id:        virttypes.EruID(status.ID),
		Running:   status.Running,
		Healthy:   status.Healthy,
		Networks:  status.Networks,
		Extension: status.Extension,
		Ttl:       ttl,

		Appname:    status.Appname,
		Entrypoint: status.Entrypoint,
		Nodename:   c.config.Hostname,
	}

	opts := &pb.SetWorkloadsStatusOptions{
		Status: []*pb.WorkloadStatus{statusPb},
	}

	var err error
	utils.WithTimeout(ctx, c.config.GlobalConnectionTimeout, func(ctx context.Context) {
		_, err = c.GetClient().SetWorkloadsStatus(ctx, opts)
	})

	if ttl == 0 {
		if err != nil {
			c.cache.Delete(status.ID)
		} else {
			c.cache.Set(status.ID, workloadStatus, getCacheTTL(c.config.HealthCheck.CacheTTL))
		}
	}

	return err
}

func (c *Store) GetWorkload(ctx context.Context, id string) (*types.Workload, error) {
	opts := &pb.WorkloadID{
		Id: virttypes.EruID(id),
	}
	wrk, err := c.GetClient().GetWorkload(ctx, opts)
	if err != nil {
		return nil, err
	}
	ans := &types.Workload{
		ID: wrk.Id,
	}
	return ans, nil
}
