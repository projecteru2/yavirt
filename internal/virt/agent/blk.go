package agent

import (
	"context"
	"regexp"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

var blkidRegex = regexp.MustCompile(`(?i)uuid="([-a-f0-9]{36})"`)

// Blkid .
func (a *Agent) Blkid(ctx context.Context, dev string) (string, error) {
	var st = <-a.ExecOutput(ctx, "blkid", dev)
	so, _, err := st.Stdio()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	var mat = blkidRegex.FindSubmatch(so)
	if mat == nil {
		return "", errors.Wrapf(terrors.ErrInvalidValue, "invalid blkid: %s", so)
	}

	return string(mat[1]), nil
}
