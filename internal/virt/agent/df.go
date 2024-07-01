package agent

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/virt/agent/types"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// output e.g., "/dev/vda1       50758780 13669928  37072468  27% /"
var dfRegex = regexp.MustCompile(`(.+?)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)%\s+(.+)`)

// GetDiskfree .
func (a *Agent) GetDiskfree(ctx context.Context, mnt string) (*types.Diskfree, error) {
	st := <-a.ExecOutput(ctx, "df", "-k", mnt)
	so, _, err := st.Stdio()
	if err != nil {
		return nil, errors.Wrapf(err, "df %s failed", mnt)
	}
	return a.parseDiskfree(string(so))
}

func (a *Agent) parseDiskfree(so string) (*types.Diskfree, error) {
	so = strings.Trim(so, " \n")
	_, line := utils.PartRight(so, "\n")

	fields := dfRegex.FindStringSubmatch(line)
	if len(fields) != 7 {
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "invalid df: %s", so)
	}

	df := &types.Diskfree{
		So:         so,
		Filesystem: fields[1],
		Mount:      fields[6],
	}
	df.Blocks, _ = utils.Atoi64(fields[2])
	df.UsedBlocks, _ = utils.Atoi64(fields[3])
	df.AvailableBlocks, _ = utils.Atoi64(fields[4])
	df.UsedPercent, _ = strconv.Atoi(fields[5])

	return df, nil
}
