package utils

import (
	"context"
	"sync"

	"github.com/alphadose/haxmap"
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/pkg/utils"
)

var ErrTooManyWatchers = errors.New("too many watchers")

type Watchers struct {
	index  utils.AtomicInt64
	wchMap *haxmap.Map[int64, *Watcher]
	events chan types.Event

	done struct {
		sync.Once
		C chan struct{}
	}
}

func NewWatchers() *Watchers {
	ws := &Watchers{
		events: make(chan types.Event),
		wchMap: haxmap.New[int64, *Watcher](),
	}
	ws.done.C = make(chan struct{})
	return ws
}

func (ws *Watchers) Len() int {
	return int(ws.wchMap.Len())
}

func (ws *Watchers) Stop() {
	defer ws.done.Do(func() {
		close(ws.done.C)
	})

	ws.wchMap.ForEach(func(_ int64, v *Watcher) bool {
		v.Stop()
		return true
	})
}

func (ws *Watchers) Run(ctx context.Context) {
	defer log.Infof(ctx, "watchers loop has done")

	for {
		select {
		case event := <-ws.events:
			ws.Notify(event)
		case <-ws.Done():
			return
		case <-ctx.Done():
			return
		}
	}
}

func (ws *Watchers) Watched(event types.Event) {
	select {
	case ws.events <- event:
		log.Infof(context.TODO(), "marks the event %v as watched", event)

	case <-ws.Done():
		log.Infof(context.TODO(), "marks the event %v failed as the Watchers has done", event)
	}
}

func (ws *Watchers) Notify(event types.Event) {
	defer log.Infof(context.TODO(), "watchers notification has done")

	stopped := []int64{}

	ws.wchMap.ForEach(func(_ int64, wch *Watcher) bool {
		select {
		case wch.events <- event:
			// notified successfully.

		case <-ws.Done():
			// It's really rare as there isn't an explicitly ws.Stop() calling now.
			stopped = append(stopped, wch.id)

		case <-wch.Done():
			// The Watcher has been stopped.
			stopped = append(stopped, wch.id)
		}
		return true
	})

	// Reaps the watchers which have been stopped.
	for _, k := range stopped {
		ws.wchMap.Del(k)
	}
}

func (ws *Watchers) Done() <-chan struct{} {
	return ws.done.C
}

func (ws *Watchers) Get() (*Watcher, error) {
	w := NewWatcher()
	w.id = ws.index.Incr()

	ws.wchMap.Set(w.id, w)

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
