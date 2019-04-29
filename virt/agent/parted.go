package agent

import (
	"context"
	"regexp"

	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/util"
)

// output e.g., "...\nDisk /dev/vdb: 107374182400B\n..."
var printSizeRegex = regexp.MustCompile(`Disk /(?:.+?): (\d+)B`)

// Parted .
type Parted struct {
	ga  Interface
	dev string
}

// NewParted .
func NewParted(ga Interface, dev string) Parted {
	return Parted{
		ga:  ga,
		dev: dev,
	}
}

// GetSize .
func (p Parted) GetSize(ctx context.Context) (int64, error) {
	st := <-p.ga.ExecOutput(ctx, "parted", "-s", p.dev, "unit", "B", "p")
	so, _, err := st.Stdio()
	if err != nil {
		return 0, errors.Annotatef(err, "parted %s print failed", p.dev)
	}
	return p.getSize(string(so))
}

func (p Parted) getSize(so string) (int64, error) {
	mat := printSizeRegex.FindStringSubmatch(so)
	if len(mat) != 2 { //nolint
		return 0, errors.Annotatef(errors.ErrInvalidValue, "invalid parted: %s", so)
	}

	return util.Atoi64(mat[1])
}
