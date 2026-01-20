package meta

import (
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Generic .
type Generic struct {
	ID           string `json:"id,omitempty" mapstructure:"id"`
	Status       string `json:"status" mapstructure:"status"`
	CreatedTime  int64  `json:"create_time" mapstructure:"create_time"`
	UpdatedTime  int64  `json:"update_time,omitempty" mapstructure:"update_time"`
	MigratedTime int64  `json:"migrate_time,omitempty" mapstructure:"migrate_time"`

	*Ver
}

type GenericInterface interface {
	Resource
	GetID() string
	GetCreatedTime() int64
	SetStatus(st string, force bool) error
	GetStatus() string
}

func NewGeneric() *Generic {
	return &Generic{
		Status:      StatusPending,
		CreatedTime: time.Now().Unix(),
		Ver:         NewVer(),
	}
}

func (g *Generic) GetID() string {
	return g.ID
}

func (g *Generic) GetCreatedTime() int64 {
	return g.CreatedTime
}

// MetaKey .
func (g *Generic) MetaKey() string {
	return VolumeKey(g.ID)
}

// JoinVirtPath .
func (g *Generic) JoinVirtPath(elem string) string {
	return filepath.Join(configs.Conf.VirtDir, elem)
}

func (g *Generic) SetStatus(st string, force bool) error {
	if !force && !g.checkStatus(st) {
		return errors.Wrapf(terrors.ErrForwardStatus, "%s => %s", g.Status, st)
	}
	g.Status = st
	return nil
}

func (g *Generic) GetStatus() string {
	return g.Status
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
		return now == StatusStarting || now == StatusResuming || now == StatusRunning

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
