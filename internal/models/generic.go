package model

import (
	"path/filepath"
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/meta"
)

// Generic .
type Generic struct {
	ID           string `json:"id,omitempty"`
	Status       string `json:"status"`
	CreatedTime  int64  `json:"create_time"`
	UpdatedTime  int64  `json:"update_time,omitempty"`
	MigratedTime int64  `json:"migrate_time,omitempty"`

	*meta.Ver
}

func newGeneric() *Generic {
	return &Generic{
		Status:      StatusPending,
		CreatedTime: time.Now().Unix(),
		Ver:         meta.NewVer(),
	}
}

// JoinVirtPath .
func (g *Generic) JoinVirtPath(elem string) string {
	return filepath.Join(config.Conf.VirtDir, elem)
}

func (g *Generic) setStatus(st string, force bool) error {
	if !(force || g.checkStatus(st)) {
		return errors.Annotatef(errors.ErrForwardStatus, "%s => %s", g.Status, st)
	}
	g.Status = st
	return nil
}

// CheckForwardStatus .
func (g *Generic) CheckForwardStatus(next string) bool {
	return g.checkStatus(next)
}

func (g *Generic) checkStatus(next string) bool {
	var now = g.Status

	switch next {
	case now:
		// met yet.
		return true

	case StatusDestroyed:
		return now == StatusDestroying
	case StatusDestroying:
		return now == StatusStopped || now == StatusDestroyed

	case StatusStopped:
		return now == StatusStopping || now == StatusMigrating || now == StatusCaptured
	case StatusStopping:
		return now == StatusRunning || now == StatusStopped

	case StatusCapturing:
		return now == StatusCapturing || now == StatusStopped
	case StatusCaptured:
		return now == StatusCapturing

	case StatusMigrating:
		return now == StatusResizing || now == StatusStopped

	case StatusResizing:
		return now == StatusStopped || now == StatusRunning

	case StatusRunning:
		return now == StatusStarting || now == StatusResuming

	case StatusPaused:
		return now == StatusPausing
	case StatusPausing:
		return now == StatusPaused || now == StatusRunning

	case StatusResuming:
		return now == StatusPaused

	case StatusStarting:
		return now == StatusStopped || now == StatusCreating

	case StatusCreating:
		return now == StatusPending

	case StatusPending:
		return now == ""

	default:
		return false
	}
}
