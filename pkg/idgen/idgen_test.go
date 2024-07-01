package idgen

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestNewGenerator(t *testing.T) {
	for idx := 0; idx < 99999; idx++ {
		g, err := New(uint32(idx))
		assert.Nil(t, err)
		assert.True(t, g.randPrefix < 1000000)
	}
	ss := strconv.FormatInt(int64(math.MaxUint32), 32)
	assert.True(t, len(ss) < 8, true)
}

func TestNewID(t *testing.T) {
	Setup(0)
	seen := make(map[string]any)
	for i := 0; i < 400000; i++ {
		id := Next()
		_, ok := seen[id]
		assert.False(t, ok)
		seen[id] = struct{}{}
		assert.True(t, CheckID(id))
	}
}

func TestConcurrency(t *testing.T) {
	Setup(1)
	var seen sync.Map
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100000; i++ {
				id := Next()
				assert.True(t, CheckID(id))
				_, loaded := seen.LoadOrStore(id, struct{}{})
				assert.False(t, loaded)
			}
		}()
	}
	wg.Wait()
}

func TestInvalidMemberID(t *testing.T) {
	assert.NilErr(t, Setup(math.MaxUint16))
	assert.Err(t, Setup(100000))
	assert.Equal(t, fmt.Sprintf("%05d", math.MaxUint16), "65535")
}

func TestGetRandomPrefix(t *testing.T) {
	seen := map[uint32]bool{}
	conflict := 0
	size := 10000
	for idx := 0; idx < size; idx++ {
		v := getRandomUint32()
		if _, ok := seen[v]; ok {
			conflict++
		}
		seen[v] = true
	}
	conflictRatio := float64(conflict) / float64(size)
	assert.True(t, conflictRatio < 0.03)
}

func TestTimeLength(t *testing.T) {
	t1 := time.Now().Add(200 * 365 * 24 * time.Hour)
	microSec := t1.UnixMilli()
	assert.Equal(t, len(fmt.Sprintf("%d", microSec)), 13)
}
