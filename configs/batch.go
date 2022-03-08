package config

import (
	"strings"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Batch .
type Batch struct {
	Bins     []string `toml:"bins"`
	FlagFile string   `toml:"flag_file"`
	ForceOK  bool     `toml:"force_ok"`
	Timeout  Duration `toml:"timeout"`
	Retry    bool     `toml:"retry"`
	Interval Duration `toml:"interval"`
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
			return nil, errors.Annotatef(errors.ErrInvalidValue, "invalid command: %s", bin)
		case 1:
			cmds[parts[0]] = nil
		default:
			cmds[parts[0]] = parts[1:]
		}
	}

	return cmds, nil
}
