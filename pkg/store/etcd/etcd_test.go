package etcd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.etcd.io/etcd/clientv3"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestRealEtcdBacthOperate(t *testing.T) {
	if os.Getenv("REAL_TEST") != "1" {
		return
	}

	configs.Conf.EtcdEndpoints = []string{"127.0.0.1:2379"}
	etcd, err := New()
	assert.NilErr(t, err)

	var ts = strconv.FormatInt(time.Now().Unix(), 10)
	var delkey = filepath.Join("/yavirt-dev/v1/test/batchops/", ts)
	var ctx = context.Background()
	assert.NilErr(t, etcd.Create(ctx, map[string]string{delkey: ts}))

	var ops = []clientv3.Op{
		clientv3.OpPut("/yavirt-dev/v1/test/batchops/putkey0", fmt.Sprintf("%v", time.Now())),
		clientv3.OpPut("/yavirt-dev/v1/test/batchops/putkey1", fmt.Sprintf("%v", time.Now())),
		clientv3.OpDelete(delkey),
	}

	succ, err := etcd.BatchOperate(ctx, ops)
	assert.NilErr(t, err)
	assert.True(t, succ)
}

func TestRealEtcdIncrUint32(t *testing.T) {
	if os.Getenv("REAL_TEST") != "1" {
		return
	}

	configs.Conf.EtcdEndpoints = []string{"127.0.0.1:2379"}
	etcd, err := New()
	assert.NilErr(t, err)

	var key = fmt.Sprintf("/yavirt-dev/v1/hosts:%d", time.Now().UnixNano())
	var wg sync.WaitGroup
	var ch = make(chan uint32, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			var n, err = etcd.IncrUint32(context.Background(), key)
			assert.NilErr(t, err)

			ch <- n
		}(i)
	}
	wg.Wait()

	var total uint32
	for i := 0; i < 100; i++ {
		total += <-ch
	}
	// Make sure of get through every digital of [1, 100].
	assert.Equal(t, uint32(5050), total)

	var id uint32
	ver, err := etcd.Get(context.Background(), key, &id)
	assert.NilErr(t, err)
	// Wrote 100-time exactly.
	assert.Equal(t, int64(100), ver)
	assert.Equal(t, uint32(100), id)
}

func TestRealEtcdCreateSerialization(t *testing.T) {
	if os.Getenv("REAL_TEST") != "1" {
		return
	}

	configs.Conf.EtcdEndpoints = []string{"127.0.0.1:2379"}
	var etcd, err = New()
	assert.NilErr(t, err)

	var wg sync.WaitGroup
	var key = fmt.Sprintf("/yavirt-dev/v1/hosts/%d", time.Now().UnixNano())
	var okThread = -1
	var okCount utils.AtomicInt64

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
			time.Sleep(time.Microsecond * time.Duration(rnd.Int31n(1000)))

			var data = map[string]string{key: strconv.Itoa(i)}

			if err := etcd.Create(context.Background(), data); err != nil {
				t.Logf(" %d: %v ", i, errors.Stack(err))
				return
			}

			okThread = i
			okCount.Incr()
		}(i)
	}

	wg.Wait()

	assert.True(t, okThread > -1)
	assert.Equal(t, int64(1), okCount.Int64())

	t.Logf("thread %d created", okThread)
}

func TestRealEtcdUpdateSerialization(t *testing.T) {
	if os.Getenv("REAL_TEST") != "1" {
		return
	}

	configs.Conf.EtcdEndpoints = []string{"127.0.0.1:2379"}
	var etcd, err = New()
	assert.NilErr(t, err)

	var key = fmt.Sprintf("/yavirt-dev/v1/hosts/%d", time.Now().UnixNano())
	var data = map[string]string{key: "0"}
	assert.NilErr(t, etcd.Create(context.Background(), data))

	var wg sync.WaitGroup
	var vers = map[string]int64{key: 1}
	var okThread = -1
	var okCount utils.AtomicInt64

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
			time.Sleep(time.Microsecond * time.Duration(rnd.Int31n(1000)))

			var data = map[string]string{key: strconv.Itoa(i)}

			if err := etcd.Update(context.Background(), data, vers); err != nil {
				t.Logf(" %d: %v ", i, errors.Stack(err))
				return
			}

			okThread = i
			okCount.Incr()
		}(i)
	}

	wg.Wait()

	assert.True(t, okThread > -1)
	assert.Equal(t, int64(1), okCount.Int64())

	t.Logf("thread %d updated", okThread)
}
