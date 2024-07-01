package models

import (
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
)

// ForwardCreating .
func (g *Guest) ForwardCreating() error {
	return g.ForwardStatus(meta.StatusCreating, false)
}

// ForwardStarting .
func (g *Guest) ForwardStarting(force bool) error {
	return g.ForwardStatus(meta.StatusStarting, force)
}

// ForwardStopped .
func (g *Guest) ForwardStopped(force bool) error {
	return g.ForwardStatus(meta.StatusStopped, force)
}

// ForwardStopping .
func (g *Guest) ForwardStopping() error {
	return g.ForwardStatus(meta.StatusStopping, false)
}

// ForwardCaptured .
func (g *Guest) ForwardCaptured() error {
	return g.ForwardStatus(meta.StatusCaptured, false)
}

// ForwardCapturing .
func (g *Guest) ForwardCapturing() error {
	return g.ForwardStatus(meta.StatusCapturing, false)
}

// ForwardDestroying .
func (g *Guest) ForwardDestroying(force bool) error {
	return g.ForwardStatus(meta.StatusDestroying, force)
}

// ForwardRunning .
func (g *Guest) ForwardRunning() error {
	return g.ForwardStatus(meta.StatusRunning, false)
}

// ForwardPaused .
func (g *Guest) ForwardPaused() error {
	return g.ForwardStatus(meta.StatusPaused, false)
}

// ForwardPausing .
func (g *Guest) ForwardPausing() error {
	return g.ForwardStatus(meta.StatusPausing, false)
}

// ForwardResuming .
func (g *Guest) ForwardResuming() error {
	return g.ForwardStatus(meta.StatusResuming, false)
}

// ForwardResizing .
func (g *Guest) ForwardResizing() error {
	return g.ForwardStatus(meta.StatusResizing, false)
}

// ForwardMigrating .
func (g *Guest) ForwardMigrating() error {
	return g.ForwardStatus(meta.StatusMigrating, false)
}

// ForwardStatus .
func (g *Guest) ForwardStatus(st string, force bool) error {
	if err := g.SetStatus(st, force); err != nil {
		return errors.WithMessagef(err, "ForwardStatus: failed to set guest status to %s", st)
	}

	if err := g.Vols.SetStatus(st, force); err != nil {
		return errors.WithMessagef(err, "ForwardStatus: failed to set volumes status to %s", st)
	}

	var res = meta.Resources{g}
	res.Concate(g.Vols.Resources())

	return meta.Save(res)
}
