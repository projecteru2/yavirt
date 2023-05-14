package guest

import (
	"sync"

	"github.com/projecteru2/yavirt/pkg/log"
)

type Watchers struct {
	sync.Mutex

	index  utils.AtomicInt64
	ws     map[int64]*Watcher
	events <-chan types.Event
}

func (ws *Watchers) Run() {
	defer log.Info("watcher loop has done")

	for {
		select {
		case event := <-ws.events:
		}
	}
}

func (ws *Watchers) Get() (*Watcher, error) {
	w := NewWatcher()
	w.id = ws.index.Incr()
	return w, nil
}

type Watcher struct {
	id     int64
	events chan types.Event
	done   struct {
		sync.Once
		C chan struct{}
	}
}

func NewWatcher() (w *GuestWatcher) {
	w = &GuestWatcher{}
	w.done.C = make(chan struct{})
	w.events = make(chan Event)
	return
}

func (w *Watcher) Events() <-chan types.Event {
	return w.events
}

func (w *Watcher) Done() <-chan struct{} {
	return w.done.C
}

func (w *Watcher) Stop() {
	w.done.Do(func() {
		close(w.done.C)
	})
}
