package manager

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/internal/virt"
)

func TestPush(t *testing.T) {
	task := testTask(t)

	exec := testExecution(t, "guest")
	exec.push(task)

	select {
	case <-time.After(time.Second):
		assert.Fail(t, "task hasn't done")
	case <-task.done:
	}

	exec.Lock()
	defer exec.Unlock()
	assert.False(t, exec.started)
	assert.Equal(t, 0, exec.list.Len())
}

func TestPushParalleled(t *testing.T) {
	exec := testExecution(t, "guest")
	group := []chan struct{}{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		task := testTask(t)
		group = append(group, task.done)

		wg.Add(1)
		go func() {
			defer wg.Done()
			exec.push(task)
		}()
	}
	wg.Wait()

	for i, done := range group {
		select {
		case <-time.After(time.Second):
			assert.Fail(t, "task %d hasn't done", i)
		case <-done:
		}
	}

	exec.Lock()
	defer exec.Unlock()
	assert.False(t, exec.started)
	assert.Equal(t, 0, exec.list.Len())
}

func testExecution(t *testing.T, id string) *execution {
	return newExecution(id)
}

func testTask(t *testing.T) *task {
	task := &task{
		id:   "test",
		op:   destroyOp,
		done: make(chan struct{}),
		ctx:  virt.NewContext(context.Background(), nil),
	}
	task.do = func(ctx virt.Context) (interface{}, error) {
		return "successful", nil
	}
	return task
}
