package manager

import (
	"fmt"
	"sync"

	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/pkg/errors"
)

const (
	destroyOp         op = "destroy"
	shutOp            op = "shutdown"
	bootOp            op = "boot"
	createOp          op = "create"
	resizeOp          op = "resize"
	miscOp            op = "misc"
	createSnapshotOp  op = "create-snapshot"
	commitSnapshotOp  op = "commit-snapshot"
	restoreSnapshotOp op = "restore-snapshot"
)

type op string

func (op op) String() string {
	return string(op)
}

type task struct {
	id     string
	op     op
	do     func(virt.Context) (any, error)
	result any
	ctx    virt.Context
	done   chan struct{}
	once   sync.Once
	err    error
}

// String .
func (t *task) String() string {
	return fmt.Sprintf("<%s, %s>", t.op, t.id)
}

func (t *task) abort() {
	t.finish()
	t.err = errors.Trace(errors.ErrSerializedTaskAborted)
}

// terminate forcibly terminates a task.
func (t *task) terminate() { //nolint
	// TODO
}

func (t *task) run() error {
	defer t.finish()

	select {
	case <-t.ctx.Done():
		if err := t.ctx.Err(); err != nil {
			t.err = err
		}
	default:
		t.result, t.err = t.do(t.ctx)
	}

	return t.err
}

func (t *task) finish() {
	t.once.Do(func() {
		close(t.done)
	})
}

type taskNotifier struct {
	done chan struct{}
	task *task
}

func (n taskNotifier) error() error {
	return n.task.err
}

func (n taskNotifier) result() any {
	return n.task.result
}

func (n taskNotifier) terminate() { //nolint
	n.task.terminate()
}
