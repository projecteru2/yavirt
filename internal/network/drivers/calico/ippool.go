package calico

import (
	"context"

	"github.com/projectcalico/calico/libcalico-go/lib/options"
)

// PoolNames .
func (h *Driver) PoolNames() (ans []string) {
	for name := range h.poolNames {
		ans = append(ans, name)
	}
	return
}

// GetIPPoolCidr .
func (h *Driver) GetIPPoolCidr(ctx context.Context, name string) (string, error) {
	ipPool, err := h.IPPools().Get(ctx, name, options.GetOptions{})
	if err != nil {
		return "", err
	}
	return ipPool.Spec.CIDR, nil
}
