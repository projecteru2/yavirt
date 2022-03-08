package types

import (
	"context"
	"io"
	"sync"

	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
)

type Console interface {
	io.ReadWriteCloser
	Fd() int // need fd for epoll event
}

// OpenConsoleFlags .
type OpenConsoleFlags struct {
	Safe     bool
	Force    bool
	Commands []string
}

// NewOpenConsoleFlags .
func NewOpenConsoleFlags(force, safe bool) OpenConsoleFlags {
	return OpenConsoleFlags{
		Force: force,
		Safe:  safe,
	}
}

// AsLibvirtConsoleFlags .
func (f *OpenConsoleFlags) AsLibvirtConsoleFlags() (flags libvirt.DomainConsoleFlags) {
	if f.Force {
		flags |= libvirt.DomainConsoleForce
	}
	if f.Safe {
		flags |= libvirt.DomainConsoleSafe
	}
	return
}

// ConsoleState .
type ConsoleState struct {
	*sync.Cond
	opened bool
}

func newConsoleState() *ConsoleState {
	return &ConsoleState{
		Cond: sync.NewCond(&sync.Mutex{}),
	}
}

type consoleStateManager struct {
	sync.Map
}

func (m *consoleStateManager) WaitUntilConsoleOpen(ctx context.Context, id string) {
	v, _ := m.LoadOrStore(id, newConsoleState())
	consoleState, ok := v.(*ConsoleState)
	if !ok {
		log.Errorf("[ConsoleStateManager] wrong type in map")
		return
	}

	consoleState.L.Lock()
	defer consoleState.L.Unlock()
	for !consoleState.opened {
		consoleState.Wait()
	}
}

func (m *consoleStateManager) MarkConsoleOpen(ctx context.Context, id string) {
	v, _ := m.LoadOrStore(id, newConsoleState())
	consoleState, ok := v.(*ConsoleState)
	if !ok {
		log.Errorf("[ConsoleStateManager] wrong type in map")
		return
	}

	consoleState.L.Lock()
	defer consoleState.L.Unlock()
	consoleState.opened = true
	consoleState.Broadcast()
}

func (m *consoleStateManager) MarkConsoleClose(ctx context.Context, id string) {
	v, _ := m.LoadOrStore(id, newConsoleState())
	consoleState, ok := v.(*ConsoleState)
	if !ok {
		log.Errorf("[ConsoleStateManager] wrong type in map")
		return
	}

	consoleState.L.Lock()
	defer consoleState.L.Unlock()
	consoleState.opened = false
	consoleState.Broadcast()
}

// ConsoleStateManager .
var ConsoleStateManager consoleStateManager
