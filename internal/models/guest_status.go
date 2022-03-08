package model

import (
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/meta"
)

// ForwardCreating .
func (g *Guest) ForwardCreating() error {
	return g.ForwardStatus(StatusCreating, false)
}

// ForwardStarting .
func (g *Guest) ForwardStarting() error {
	return g.ForwardStatus(StatusStarting, false)
}

// ForwardStopped .
func (g *Guest) ForwardStopped(force bool) error {
	return g.ForwardStatus(StatusStopped, force)
}

// ForwardStopping .
func (g *Guest) ForwardStopping() error {
	return g.ForwardStatus(StatusStopping, false)
}

// ForwardCaptured .
func (g *Guest) ForwardCaptured() error {
	return g.ForwardStatus(StatusCaptured, false)
}

// ForwardCapturing .
func (g *Guest) ForwardCapturing() error {
	return g.ForwardStatus(StatusCapturing, false)
}

// ForwardDestroying .
func (g *Guest) ForwardDestroying(force bool) error {
	return g.ForwardStatus(StatusDestroying, force)
}

// ForwardRunning .
func (g *Guest) ForwardRunning() error {
	return g.ForwardStatus(StatusRunning, false)
}

// ForwardPaused .
func (g *Guest) ForwardPaused() error {
	return g.ForwardStatus(StatusPaused, false)
}

// ForwardPausing .
func (g *Guest) ForwardPausing() error {
	return g.ForwardStatus(StatusPausing, false)
}

// ForwardResuming .
func (g *Guest) ForwardResuming() error {
	return g.ForwardStatus(StatusResuming, false)
}

// ForwardResizing .
func (g *Guest) ForwardResizing() error {
	return g.ForwardStatus(StatusResizing, false)
}

// ForwardMigrating .
func (g *Guest) ForwardMigrating() error {
	return g.ForwardStatus(StatusMigrating, false)
}

// ForwardStatus .
func (g *Guest) ForwardStatus(st string, force bool) error {
	if err := g.setStatus(st, force); err != nil {
		return errors.Trace(err)
	}

	if err := g.Vols.setStatus(st, force); err != nil {
		return errors.Trace(err)
	}

	var res = meta.Resources{g}
	res.Concate(g.Vols.resources())

	return meta.Save(res)
}
