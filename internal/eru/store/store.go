package store

import (
	"context"

	"github.com/projecteru2/yavirt/internal/eru/types"
)

// Store wrapper of remote calls
type Store interface { //nolint
	CheckHealth(ctx context.Context) error
	GetNode(ctx context.Context, nodename string) (*types.Node, error)
	SetNodeStatus(ctx context.Context, ttl int64) error
	GetNodeStatus(ctx context.Context, nodename string) (*types.NodeStatus, error)
	AddNode(ctx context.Context, opts *types.AddNodeOpts) (*types.Node, error)
	SetNode(ctx context.Context, opts *types.SetNodeOpts) (*types.Node, error)
	SetWorkloadStatus(ctx context.Context, status *types.WorkloadStatus, ttl int64) error
	GetIdentifier(ctx context.Context) string
	NodeStatusStream(ctx context.Context) (<-chan *types.NodeStatus, <-chan error)
	ListPodNodes(ctx context.Context, all bool, podname string, labels map[string]string) ([]*types.Node, error)
	ListNodeWorkloads(ctx context.Context, nodename string) ([]*types.Workload, error)
	GetNodeResource(ctx context.Context, nodename string) (*types.NodeResource, error)
	GetWorkload(ctx context.Context, id string) (*types.Workload, error)
}
