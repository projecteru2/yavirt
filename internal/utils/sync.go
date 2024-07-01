package utils

import (
	"sync"
)

// GroupCAS indicates cas locks which are grouped by keys.
type GroupCAS struct {
	sync.Mutex
	locks map[string]struct{}
}

// NewGroupCAS .
func NewGroupCAS() *GroupCAS {
	return &GroupCAS{
		locks: map[string]struct{}{},
	}
}

// Acquire tries to acquire a cas lock.
func (g *GroupCAS) Acquire(key string) (free func(), acquired bool) {
	g.Lock()
	defer g.Unlock()
	if _, ok := g.locks[key]; ok {
		return nil, false
	}

	g.locks[key] = struct{}{}
	free = func() {
		g.Lock()
		defer g.Unlock()
		delete(g.locks, key)
	}

	return free, true
}
