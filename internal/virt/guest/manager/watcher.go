package manager

import (
	"sync"

	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

var ErrTooManyWatchers = errors.New("too many watchers")

type Watchers struct {
	sync.Mutex

	index  utils.AtomicInt64
	ws     map[int64]*Watcher
	events chan types.Event

	done struct {
		sync.Once
		C chan struct{}
	}
}

func NewWatchers() *Watchers {
	ws := &Watchers{
		events: make(chan types.Event),
		ws:     map[int64]*Watcher{},
	}
	ws.done.C = make(chan struct{})
	return ws
}

func (ws *Watchers) Stop() {
	defer ws.done.Do(func() {
		close(ws.done.C)
	})

	ws.Lock()
	defer ws.Unlock()

	for _, w := range ws.ws {
		w.Stop()
	}
}

func (ws *Watchers) Run() {
	defer log.Infof("watchers loop has done")

	for {
		select {
		case event := <-ws.events:
			ws.Notify(event)

		case <-ws.Done():
			return
		}
	}
}

func (ws *Watchers) Watched(event types.Event) {
	select {
	case ws.events <- event:
		log.Infof("marks the event %v as watched", event)

	case <-ws.Done():
		log.Infof("marks the event %v failed as the Watchers has done", event)
	}
}

func (ws *Watchers) Notify(event types.Event) {
	defer log.Infof("watchers notification has done")

	stopped := []int64{}

	ws.Lock()
	defer ws.Unlock()

	for _, w := range ws.ws {
		select {
		case w.events <- event:
			// notified successfully.

		case <-ws.Done():
			// It's really rare as there isn't an explicitly ws.Stop() calling now.
			stopped = append(stopped, w.id)

		case <-w.Done():
			// The Watcher has been stopped.
			stopped = append(stopped, w.id)
		}
	}

	// Reaps the watchers which have been stopped.
	for _, k := range stopped {
		delete(ws.ws, k)
	}
}

func (ws *Watchers) Done() <-chan struct{} {
	return ws.done.C
}

func (ws *Watchers) Get() (*Watcher, error) {
	ws.Lock()
	defer ws.Unlock()

	w := NewWatcher()
	w.id = ws.index.Incr()

	ws.ws[w.id] = w

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

func NewWatcher() (w *Watcher) {
	w = &Watcher{}
	w.done.C = make(chan struct{})
	w.events = make(chan types.Event)
	return
}

func (w *Watcher) ID() int64 {
	return w.id
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
