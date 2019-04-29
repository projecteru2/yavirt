package util

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/test/assert"
)

func TestFlock(t *testing.T) {
	var rand = strconv.FormatInt(time.Now().UnixNano(), 10)

	var fpth = filepath.Join(os.TempDir(), rand)
	defer os.Remove(fpth)

	var flocks [64]*Flock
	var wg sync.WaitGroup
	var errCnt, okCnt AtomicInt64
	var n = 64

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			flocks[i] = NewFlock(fpth)
			if err := flocks[i].Trylock(); err != nil {
				errCnt.Incr()
			} else {
				okCnt.Incr()
			}
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int64(1), okCnt.Int64())
	assert.Equal(t, int64(n-1), errCnt.Int64())

	for i := 0; i < n; i++ {
		flocks[i].Unlock()
	}

	assert.NilErr(t, NewFlock(fpth).Trylock())
}

func TestFlockSingle(t *testing.T) {
	var rand = strconv.FormatInt(time.Now().UnixNano(), 10)

	var fpth = filepath.Join(os.TempDir(), rand)
	defer os.Remove(fpth)

	var flock = NewFlock(fpth)
	defer flock.Close()

	var wg sync.WaitGroup

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NilErr(t, flock.Trylock())
		}()
	}
	wg.Wait()
}
