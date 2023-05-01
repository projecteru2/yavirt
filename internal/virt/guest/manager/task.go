package manager

import (
	"fmt"
	"sync"

	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/pkg/errors"
)

const (
	destroyOp op = iota
	shutOp
	bootOp
	createOp
	resizeOp
	miscOp
	createSnapshotOp
	commitSnapshotOp
	restoreSnapshotOp
)

type op int

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
	return fmt.Sprintf("<%s, %s>", t.descOp(), t.id)
}

func (t *task) descOp() string {
	switch t.op {
	case destroyOp:
		return "destroy"
	case shutOp:
		return "shutdown"
	case bootOp:
		return "boot"
	case createOp:
		return "create"
	case resizeOp:
		return "resize"
	case miscOp:
		return "misc"
	case createSnapshotOp:
		return "create-snapshot"
	case commitSnapshotOp:
		return "commit-snapshot"
	case restoreSnapshotOp:
		return "restore-snapshot"
	default:
		return "unknown"
	}
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
