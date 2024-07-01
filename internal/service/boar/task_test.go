package boar

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestTask(t *testing.T) {
	var counter int32
	timeout := 3 * time.Second
	start := time.Now()
	p, err := newTaskPool(100)
	assert.Nil(t, err)
	tsk := newTask(context.Background(), "test", types.OpStart, func(ctx context.Context) (any, error) {
		atomic.AddInt32(&counter, 1)
		time.Sleep(timeout)
		return nil, nil
	})
	p.SubmitTask(tsk)
	<-tsk.Done()
	assert.Equal(t, int32(1), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout)
	assert.True(t, time.Since(start) < timeout+3*time.Second)
}

func TestSingleTypeTask(t *testing.T) {
	var counter int32
	timeout := 3 * time.Second
	nTasks := 4
	tasks := make([]*task, nTasks)
	start := time.Now()
	p, err := newTaskPool(100)
	assert.Nil(t, err)

	for i := 0; i < nTasks; i++ {
		tsk := newTask(context.Background(), "test", types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(timeout)
			return nil, nil
		})
		p.SubmitTask(tsk)
		tasks[i] = tsk
	}
	for i := 0; i < nTasks; i++ {
		<-tasks[i].Done()
	}
	assert.Equal(t, int32(nTasks), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout*time.Duration(nTasks))
	assert.True(t, time.Since(start) < timeout*time.Duration(nTasks)+3*time.Second)
}

func TestSingleTypeTaskSeq(t *testing.T) {
	var counter int32
	timeout := 3 * time.Second
	nTasks := 4
	tasks := make([]*task, nTasks)
	start := time.Now()
	p, err := newTaskPool(100)
	assert.Nil(t, err)

	for i := 0; i < nTasks; i++ {
		tsk := newTask(context.Background(), "test", types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(timeout)
			return nil, nil
		})
		p.SubmitTask(tsk)
		tasks[i] = tsk
		<-tasks[i].Done()
	}
	assert.Equal(t, int32(nTasks), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout*time.Duration(nTasks))
	assert.True(t, time.Since(start) < timeout*time.Duration(nTasks)+3*time.Second)
}

func TestSingleTypeFailedTask(t *testing.T) {
	var counter int32
	timeout := 3 * time.Second
	nTasks := 3
	nFailed := 4
	tasks := make([]*task, nTasks+nFailed)
	start := time.Now()
	p, err := newTaskPool(100)
	assert.Nil(t, err)

	for i := 0; i < nTasks; i++ {
		tsk := newTask(context.Background(), "test", types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(timeout)
			return nil, nil
		})
		p.SubmitTask(tsk)
		tasks[i] = tsk
	}

	for i := 0; i < nFailed; i++ {
		tsk := newTask(context.Background(), "test", types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			return nil, fmt.Errorf("failed")
		})
		p.SubmitTask(tsk)
		tasks[i+nTasks] = tsk
	}
	for i := 0; i < len(tasks); i++ {
		<-tasks[i].Done()
		_, err := tasks[i].result()
		if i < nTasks {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
	assert.Equal(t, int32(nTasks+1), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout*time.Duration(nTasks))
	assert.True(t, time.Since(start) < timeout*time.Duration(nTasks)+3*time.Second)
}

func TestMultiTypeTask(t *testing.T) {
	var counter int32
	timeout := 3 * time.Second
	nTasks := 4
	tasks := make([]*task, nTasks)
	start := time.Now()
	p, err := newTaskPool(100)
	assert.Nil(t, err)

	for i := 0; i < nTasks; i++ {
		id := fmt.Sprintf("test%d", i)
		tsk := newTask(context.Background(), id, types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(timeout)
			return nil, nil
		})
		p.SubmitTask(tsk)
		tasks[i] = tsk
	}
	for i := 0; i < nTasks; i++ {
		<-tasks[i].Done()
	}
	assert.Equal(t, int32(nTasks), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout)
	assert.True(t, time.Since(start) < timeout+3*time.Second)
}

func TestSmallPool(t *testing.T) {
	var counter int32
	poolSize := 2
	timeout := 3 * time.Second
	nTasks := 4
	tasks := make([]*task, nTasks)
	start := time.Now()
	p, err := newTaskPool(poolSize)
	assert.Nil(t, err)

	for i := 0; i < nTasks; i++ {
		id := fmt.Sprintf("test%d", i)
		tsk := newTask(context.Background(), id, types.OpStart, func(ctx context.Context) (any, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(timeout)
			return nil, nil
		})
		err := p.SubmitTask(tsk)
		if i < poolSize {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
			tsk.finish()
		}
		tasks[i] = tsk
	}
	for i := 0; i < nTasks; i++ {
		<-tasks[i].Done()
	}
	assert.Equal(t, int32(poolSize), atomic.LoadInt32(&counter))
	assert.True(t, time.Since(start) > timeout)
	assert.True(t, time.Since(start) < timeout+3*time.Second)
}
