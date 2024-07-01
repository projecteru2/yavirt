package core

import (
	"context"
	"sync"
	"time"

	"github.com/projecteru2/core/client"
	pb "github.com/projecteru2/core/rpc/gen"
	coretypes "github.com/projecteru2/core/types"
	"github.com/projecteru2/yavirt/internal/eru/types"

	"github.com/patrickmn/go-cache"
	"github.com/projecteru2/core/log"
)

// Store use core to store meta
type Store struct {
	clientPool *client.Pool
	config     *types.Config
	cache      *cache.Cache
}

var coreStore *Store
var once sync.Once

// New new a Store
func New(ctx context.Context, config *types.Config) (*Store, error) {
	auth := coretypes.AuthConfig{
		Username: config.Username,
		Password: config.Password,
	}
	clientPoolConfig := &client.PoolConfig{
		EruAddrs:          config.Addrs,
		Auth:              auth,
		ConnectionTimeout: config.GlobalConnectionTimeout,
	}
	clientPool, err := client.NewCoreRPCClientPool(ctx, clientPoolConfig)
	if err != nil {
		return nil, err
	}
	cache := cache.New(time.Duration(config.HealthCheck.CacheTTL)*time.Second, 24*time.Hour)
	return &Store{clientPool, config, cache}, nil
}

// GetClient returns a gRPC client
func (c *Store) GetClient() pb.CoreRPCClient {
	return c.clientPool.GetClient()
}

// Init inits the core store only once
func Init(ctx context.Context, config *types.Config) {
	once.Do(func() {
		var err error
		coreStore, err = New(ctx, config)
		if err != nil {
			log.WithFunc("core.client").Error(ctx, err, "failed to create core store")
			return
		}
	})
}

// Get returns the core store instance
func Get() *Store {
	return coreStore
}

func (c *Store) CheckHealth(ctx context.Context) error {
	_, err := c.GetClient().Info(ctx, &pb.Empty{})
	return err
}
