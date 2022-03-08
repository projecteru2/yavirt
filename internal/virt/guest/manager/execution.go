package manager

import (
	"container/list"
	"sync"

	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/pkg/log"
)

type execution struct {
	sync.Mutex
	list    *list.List
	id      string
	started bool
	exit    chan struct{}
}

func newExecution(id string) *execution {
	return &execution{
		list: list.New(),
		id:   id,
		exit: make(chan struct{}),
	}
}

func (e *execution) push(t *task) {
	log.Debugf("ready to push task %s", t)
	if e.start(t) {
		log.Debugf("execution %s has been started yet for the task %s", e.id, t)
		return
	}
	log.Debugf("ready to start execution %s for the task %s", e.id, t)

	go func() {
		defer log.Debugf("execution %s is exiting", e.id)

		for {
			t := e.readTask()
			if t == nil {
				return
			}

			if err := t.run(); err != nil {
				log.ErrorStack(err)
				metrics.IncrError()

				e.Lock()
				defer e.Unlock()

				if e.list.Len() > 0 {
					log.Warnf("execution %s list is not empty, but the task %s had an error", e.id, t)

					// Notifies there's an error, and removes all element.
					for {
						cur := e.list.Front()
						if cur == nil {
							log.Warnf("execution %s list elements were notified and cleared", e.id)
							break
						}
						e.list.Remove(cur)

						t := cur.Value.(*task)
						t.abort()
					}
				}

				e.started = false
				return
			}
		}
	}()
}

func (e *execution) readTask() (t *task) {
	e.Lock()
	defer e.Unlock()

	defer func() {
		if t == nil {
			e.started = false
		}
	}()

	cur := e.list.Front()
	if cur == nil {
		log.Debugf("execution %s list is empty", e.id)
		return
	}

	e.list.Remove(cur)

	// TODO: process exit signal.
	select {
	case <-e.exit:
		log.Warnf("execution %s had received an exit signal", e.id)
	default:
	}

	return cur.Value.(*task)
}

func (e *execution) start(t *task) bool {
	e.Lock()
	defer e.Unlock()

	e.list.PushBack(t)

	if e.started {
		return true
	}

	e.started = true
	return false
}
