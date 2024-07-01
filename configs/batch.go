package configs

import (
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Batch .
type Batch struct {
	Bins     []string      `toml:"bins"`
	FlagFile string        `toml:"flag_file"`
	ForceOK  bool          `toml:"force_ok"`
	Timeout  time.Duration `toml:"timeout"`
	Retry    bool          `toml:"retry"`
	Interval time.Duration `toml:"interval"`
}

// IsRunOnce .
func (b Batch) IsRunOnce() bool {
	return len(b.FlagFile) > 0
}

// GetCommands .
func (b Batch) GetCommands() (map[string][]string, error) {
	var cmds = map[string][]string{}

	for _, bin := range b.Bins {
		switch parts := strings.Split(bin, " "); len(parts) {
		case 0:
			return nil, errors.Wrapf(terrors.ErrInvalidValue, "invalid command: %s", bin)
		case 1:
			cmds[parts[0]] = nil
		default:
			cmds[parts[0]] = parts[1:]
		}
	}

	return cmds, nil
}
