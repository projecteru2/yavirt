package core

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/projecteru2/core/log"
	resourcetypes "github.com/projecteru2/core/resource/types"
	pb "github.com/projecteru2/core/rpc/gen"
	"github.com/projecteru2/yavirt/internal/eru/types"
	"github.com/projecteru2/yavirt/internal/utils"
	"github.com/samber/lo"
)

// GetNode return a node by core
func (c *Store) GetNode(ctx context.Context, nodename string) (*types.Node, error) {
	var resp *pb.Node
	var err error

	utils.WithTimeout(ctx, c.config.GlobalConnectionTimeout, func(ctx context.Context) {
		resp, err = c.GetClient().GetNode(ctx, &pb.GetNodeOptions{Nodename: nodename})
	})

	if err != nil {
		return nil, err
	}

	node := &types.Node{
		Name:      resp.Name,
		Podname:   resp.Podname,
		Endpoint:  resp.Endpoint,
		Available: resp.Available,
	}
	return node, nil
}

// SetNodeStatus reports the status of node
// SetNodeStatus always reports alive status,
// when not alive, TTL will cause expiration of node
func (c *Store) SetNodeStatus(ctx context.Context, ttl int64) error {
	opts := &pb.SetNodeStatusOptions{
		Nodename: c.config.Hostname,
		Ttl:      ttl,
	}
	var err error
	utils.WithTimeout(ctx, c.config.GlobalConnectionTimeout, func(ctx context.Context) {
		_, err = c.GetClient().SetNodeStatus(ctx, opts)
	})

	return err
}

// GetNodeStatus gets the status of node
func (c *Store) GetNodeStatus(ctx context.Context, nodename string) (*types.NodeStatus, error) {
	var resp *pb.NodeStatusStreamMessage
	var err error

	utils.WithTimeout(ctx, c.config.GlobalConnectionTimeout, func(ctx context.Context) {
		resp, err = c.GetClient().GetNodeStatus(ctx, &pb.GetNodeStatusOptions{Nodename: nodename})
	})

	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		err = errors.New(resp.Error)
	}

	status := &types.NodeStatus{
		Nodename: resp.Nodename,
		Podname:  resp.Podname,
		Alive:    resp.Alive,
		Error:    err,
	}
	return status, nil
}

// NodeStatusStream watches the changes of node status
func (c *Store) NodeStatusStream(ctx context.Context) (<-chan *types.NodeStatus, <-chan error) {
	msgChan := make(chan *types.NodeStatus)
	errChan := make(chan error)

	_ = utils.Pool.Submit(func() {
		defer close(msgChan)
		defer close(errChan)

		client, err := c.GetClient().NodeStatusStream(ctx, &pb.Empty{})
		if err != nil {
			errChan <- err
			return
		}

		for {
			message, err := client.Recv()
			if err != nil {
				errChan <- err
				return
			}
			nodeStatus := &types.NodeStatus{
				Nodename: message.Nodename,
				Podname:  message.Podname,
				Alive:    message.Alive,
				Error:    nil,
			}
			if message.Error != "" {
				nodeStatus.Error = errors.New(message.Error)
			}
			msgChan <- nodeStatus
		}
	})

	return msgChan, errChan
}

// ListPodNodes list nodes by given conditions, note that not all the fields are filled.
func (c *Store) ListPodNodes(ctx context.Context, all bool, podname string, labels map[string]string) ([]*types.Node, error) {
	ch, err := c.listPodeNodes(ctx, &pb.ListNodesOptions{
		Podname: podname,
		All:     all,
		Labels:  labels,
	})
	if err != nil {
		return nil, err
	}

	nodes := []*types.Node{}
	for n := range ch {
		nodes = append(nodes, &types.Node{
			Name:     n.Name,
			Endpoint: n.Endpoint,
			Podname:  n.Podname,
			Labels:   n.Labels,
		})
	}
	return nodes, nil
}

func (c *Store) listPodeNodes(ctx context.Context, opt *pb.ListNodesOptions) (ch chan *pb.Node, err error) {
	ch = make(chan *pb.Node)

	utils.WithTimeout(ctx, c.config.GlobalConnectionTimeout, func(ctx context.Context) {
		var stream pb.CoreRPC_ListPodNodesClient
		if stream, err = c.GetClient().ListPodNodes(ctx, opt); err != nil {
			return
		}

		_ = utils.Pool.Submit(func() {
			defer close(ch)
			for {
				node, err := stream.Recv()
				if err != nil {
					if err != io.EOF { //nolint:nolintlint
						log.WithFunc("listPodeNodes").Error(ctx, err, "get node stream failed")
					}
					return
				}
				ch <- node
			}
		})
	})

	return ch, nil
}

func (c *Store) ListNodeWorkloads(ctx context.Context, nodename string) ([]*types.Workload, error) {
	opts := &pb.GetNodeOptions{
		Nodename: nodename,
	}
	wrks, err := c.GetClient().ListNodeWorkloads(ctx, opts)
	if err != nil {
		return nil, err
	}
	ans := lo.Map(wrks.Workloads, func(w *pb.Workload, _ int) *types.Workload {
		return &types.Workload{
			ID: w.Id,
		}
	})
	return ans, nil
}

func (c *Store) GetNodeResource(ctx context.Context, nodename string) (*types.NodeResource, error) {
	resp, err := c.GetClient().GetNodeResource(ctx, &pb.GetNodeResourceOptions{
		Opts: &pb.GetNodeOptions{
			Nodename: nodename,
		},
	})
	if err != nil {
		return nil, err
	}
	capacity := resourcetypes.Resources{}
	if err = json.Unmarshal([]byte(resp.ResourceCapacity), &capacity); err != nil {
		return nil, err
	}
	return &types.NodeResource{
		Capacity: capacity,
	}, nil
}

func (c *Store) SetNode(ctx context.Context, opts *types.SetNodeOpts) (*types.Node, error) {
	resp, err := c.GetClient().SetNode(ctx, &pb.SetNodeOptions{
		Nodename:      opts.Nodename,
		Endpoint:      opts.Endpoint,
		Ca:            opts.Ca,
		Cert:          opts.Cert,
		Key:           opts.Key,
		Labels:        opts.Labels,
		Resources:     opts.Resources,
		Delta:         opts.Delta,
		WorkloadsDown: opts.WorkloadsDown,
	})
	if err != nil {
		return nil, err
	}
	return &types.Node{
		Name:      resp.Name,
		Podname:   resp.Podname,
		Endpoint:  resp.Endpoint,
		Available: resp.Available,
	}, nil
}

func (c *Store) AddNode(ctx context.Context, opts *types.AddNodeOpts) (*types.Node, error) {
	resp, err := c.GetClient().AddNode(ctx, &pb.AddNodeOptions{
		Nodename:  opts.Nodename,
		Endpoint:  opts.Endpoint,
		Podname:   opts.Podname,
		Ca:        opts.Ca,
		Cert:      opts.Cert,
		Key:       opts.Key,
		Labels:    opts.Labels,
		Resources: opts.Resources,
	})
	if err != nil {
		return nil, err
	}
	return &types.Node{
		Name:      resp.Name,
		Podname:   resp.Podname,
		Endpoint:  resp.Endpoint,
		Available: resp.Available,
	}, nil
}
