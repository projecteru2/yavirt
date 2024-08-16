package agent

import (
	"context"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/virt/agent/types"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

var blkidRegex = regexp.MustCompile(`(?i)uuid="([-a-f0-9]{36}).*TYPE="([^"]+)"`)

// Blkid .
func (a *Agent) Blkid(ctx context.Context, dev string) (*types.BlkidInfo, error) {
	var st = <-a.ExecOutput(ctx, "blkid", dev)
	so, _, err := st.Stdio()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var mat = blkidRegex.FindSubmatch(so)
	if mat == nil {
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "invalid blkid: %s", so)
	}

	return &types.BlkidInfo{
		ID:   string(mat[1]),
		Type: string(mat[2]),
	}, nil
}
