package ipam

import (
	"context"

	"github.com/projecteru2/yavirt/meta"
)

// Ipam .
type Ipam interface {
	Assign(ctx context.Context) (meta.IP, error)
	Release(context.Context, ...meta.IP) error
	Query(context.Context, meta.IPNets) ([]meta.IP, error)
}
