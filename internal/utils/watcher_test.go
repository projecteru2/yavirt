package utils

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestWatchers_StopSingleOne(t *testing.T) {
	ws := NewWatchers()
	defer ws.Stop()

	go ws.Run(context.Background())

	var wg sync.WaitGroup
	defer wg.Wait()

	watcher, err := ws.Get()
	assert.NilErr(t, err)
	assert.Equal(t, int64(1), watcher.id)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-watcher.Events():
			case <-watcher.Done():
				return
			}
		}
	}()

	watcher.Stop()
}

func TestWatchers_StopAll(t *testing.T) {
	ws := NewWatchers()

	// WaitGroup of all watchers have done.
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.Run(context.Background())
	}()

	var sum utils.AtomicInt64
	n := 10

	// WaitGroup of all watchers have been registered.
	var rwg sync.WaitGroup

	for i := 0; i < n; i++ {
		rwg.Add(1)
		go func() {
			defer rwg.Done()

			watcher, err := ws.Get()
			assert.NilErr(t, err)
			assert.True(t, watcher.id > 0)

			sum.Add(watcher.id)

			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					select {
					case <-watcher.Events():
					case <-watcher.Done():
						return
					}
				}
			}()
		}()
	}

	// Waiting for all watchers have been registered.
	rwg.Wait()

	ws.Stop()

	assert.Equal(t, int64(n*(n+1)/2), sum.Int64())
}

func TestWatchers_WatchedEvent(t *testing.T) {
	ws := NewWatchers()

	// WaitGroup of all watchers have done.
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.Run(context.Background())
	}()

	var sum, recv utils.AtomicInt64
	n := 10

	watchersCount := n * 2

	// WaitGroup of all watchers have been registered.
	var rwg sync.WaitGroup

	for i := 0; i < watchersCount; i++ {
		rwg.Add(1)
		go func() {
			defer rwg.Done()

			watcher, err := ws.Get()
			assert.NilErr(t, err)
			assert.True(t, watcher.id > 0)

			sum.Add(watcher.id)

			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					select {
					case event := <-watcher.Events():
						v, err := strconv.Atoi(event.Op.String())
						assert.NilErr(t, err)

						recv.Add(int64(v))

					case <-watcher.Done():
						return
					}
				}
			}()

			if watcher.id%2 == 0 {
				watcher.Stop()
			}
		}()
	}

	// Waiting for all watchers have been registered.
	rwg.Wait()

	// WaitGroup of all Watched callings have done.
	var nwg sync.WaitGroup
	for i := 1; i <= n; i++ {
		nwg.Add(1)
		go func(action string) {
			defer nwg.Done()
			ws.Watched(types.Event{
				Op: types.Operator(action),
			})
		}(strconv.Itoa(i))
	}
	nwg.Wait()

	retries := 3
	expRecv := int64(n * (n + 1) / 2 * n)
	expSum := int64(watchersCount * (watchersCount + 1) / 2)
	for i := 0; i < retries; i++ {
		if expSum == sum.Int64() && expRecv == recv.Int64() {
			break
		}
		time.Sleep(time.Millisecond * time.Duration((i+1)*100))
	}
	assert.Equal(t, expSum, sum.Int64())
	// There were n watchers which received n events.
	assert.Equal(t, expRecv, recv.Int64())

	ws.Stop()
}
