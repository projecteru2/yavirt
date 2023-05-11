package calico

import (
	"context"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

// PoolNames .
func (h *Handler) PoolNames() []string {
	return h.poolNames
}

// GetIPPoolCidr .
func (h *Handler) GetIPPoolCidr(ctx context.Context, name string) (string, error) {
	ipPool, err := h.cali.IPPools().Get(ctx, name, options.GetOptions{})
	if err != nil {
		return "", err
	}
	return ipPool.Spec.CIDR, nil
}
