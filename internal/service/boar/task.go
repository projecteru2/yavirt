package boar

import (
	"context"
	"fmt"
	"sync"

	llq "github.com/emirpasic/gods/queues/linkedlistqueue"
	"github.com/panjf2000/ants/v2"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/types"
)

type taskNotifier struct {
	id  string
	err error
}

type task struct {
	mu sync.Mutex

	id   string
	op   types.Operator
	ctx  context.Context
	do   func(context.Context) (any, error)
	res  any
	err  error
	done struct {
		once sync.Once
		c    chan struct{}
	}
}

func newTask(ctx context.Context, id string, op types.Operator, fn doFunc) *task {
	t := &task{
		id:  id,
		op:  op,
		ctx: ctx,
		do:  fn,
	}
	t.done.c = make(chan struct{})
	return t
}

// String .
func (t *task) String() string {
	return fmt.Sprintf("%s <%s>", t.id, t.op)
}

func (t *task) Done() <-chan struct{} {
	return t.done.c
}

func (t *task) run(ctx context.Context) error {
	defer t.finish()

	var (
		res any
		err error
	)

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		res, err = t.do(ctx)
	}
	t.setResult(res, err)
	return err
}

func (t *task) finish() {
	t.done.once.Do(func() {
		close(t.done.c)
	})
}

func (t *task) result() (any, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.res, t.err
}

func (t *task) setResult(res any, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.res, t.err = res, err
}

// all tasks in a task queue have same guest id.
type taskQueue struct {
	*llq.Queue
	id string
}

func newTaskQueue(id string) *taskQueue {
	return &taskQueue{
		Queue: llq.New(),
		id:    id,
	}
}

func (tq *taskQueue) revertAll(err error) {
	for {
		obj, ok := tq.Dequeue()
		if !ok {
			break
		}
		t, _ := obj.(*task)
		t.finish()
		t.setResult(nil, err)

	}
}

type taskPool struct {
	mu sync.Mutex
	// pool size
	size int
	// key is guest id
	mgr      map[string]*taskQueue
	pool     *ants.Pool
	notifier chan taskNotifier
}

func newTaskPool(max int) (*taskPool, error) {
	p, err := ants.NewPool(max, ants.WithNonblocking(true))
	if err != nil {
		return nil, err
	}
	tp := &taskPool{
		size:     max,
		pool:     p,
		mgr:      make(map[string]*taskQueue),
		notifier: make(chan taskNotifier, 10),
	}
	go tp.loop()
	return tp, nil
}

func (p *taskPool) SubmitTask(t *task) (err error) {
	needNotify := false
	err = p.withLocker(func() error {
		if _, ok := p.mgr[t.id]; !ok {
			if p.size > 0 && len(p.mgr) >= p.size {
				return fmt.Errorf("task pool is full")
			}
			p.mgr[t.id] = newTaskQueue(t.id)
			needNotify = true
		}
		p.mgr[t.id].Enqueue(t)
		return nil
	})
	if err != nil {
		return err
	}
	if needNotify {
		p.notifier <- taskNotifier{
			id:  t.id,
			err: nil,
		}
	}
	return
}

func (p *taskPool) loop() {
	logger := log.WithFunc("taskPool.loop")
	for v := range p.notifier {
		var t *task
		_ = p.withLocker(func() error {
			if v.err != nil {
				// when error, revert all tasks in the same queue
				tq := p.mgr[v.id]
				tq.revertAll(v.err)
			}
			tq := p.mgr[v.id]
			if tq.Empty() {
				delete(p.mgr, v.id)
				return nil
			}
			obj, _ := tq.Dequeue()
			t, _ = obj.(*task)
			return nil
		})

		if t == nil {
			continue
		}
		err := p.pool.Submit(func() {
			if err := t.run(t.ctx); err != nil {
				logger.Error(context.TODO(), err)
			}
			_, err := t.result()
			p.notifier <- taskNotifier{
				id:  t.id,
				err: err,
			}
		})
		if err != nil {
			// the pool is full, it never happens, because when submitting task, the size is already checked
			logger.Errorf(context.TODO(), err, "BUG: failed to submit task<%s> %s", t.id, err)
		}
	}
}

func (p *taskPool) withLocker(f func() error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return f()
}

func (p *taskPool) Release() {
	p.pool.Release()
}
