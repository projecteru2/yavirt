package etcd

import (
	"context"
	"crypto/tls"
	"strconv"
	"sync"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/transport"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/store/etcd/embedded"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// ETCDClientV3 .
type ETCDClientV3 interface { //nolint
	clientv3.KV
	clientv3.Lease
	clientv3.Watcher
}

// ETCD .
type ETCD struct {
	sync.Mutex
	cliv3 ETCDClientV3
	cfg   configs.ETCDConfig
}

// New .
func New(cfg configs.ETCDConfig, t *testing.T) (*ETCD, error) {
	var cliv3 *clientv3.Client
	var err error
	var tlsConfig *tls.Config

	switch {
	case t != nil:
		embededETCD := embedded.NewCluster(t, cfg.Prefix)
		cliv3 = embededETCD.RandClient()
		log.Infof(context.TODO(), "use embedded cluster")
	default:
		if utils.FileExists(cfg.CA) && utils.FileExists(cfg.Key) && utils.FileExists(cfg.Cert) {
			tlsInfo := transport.TLSInfo{
				TrustedCAFile: cfg.CA,
				KeyFile:       cfg.Key,
				CertFile:      cfg.Cert,
			}
			tlsConfig, err = tlsInfo.ClientConfig()
			if err != nil {
				return nil, err
			}
		}
		if cliv3, err = clientv3.New(clientv3.Config{
			Endpoints: cfg.Endpoints,
			Username:  cfg.Username,
			Password:  cfg.Password,
			TLS:       tlsConfig,
		}); err != nil {
			return nil, err
		}
		// cliv3.KV = namespace.NewKV(cliv3.KV, config.Prefix)
		// cliv3.Watcher = namespace.NewWatcher(cliv3.Watcher, config.Prefix)
		// cliv3.Lease = namespace.NewLease(cliv3.Lease, config.Prefix)
	}
	return &ETCD{cliv3: cliv3, cfg: cfg}, nil
}

// IncrUint32 .
func (e *ETCD) IncrUint32(ctx context.Context, key string) (n uint32, err error) {
	var mutex utils.Locker
	if mutex, err = e.NewMutex(key); err != nil {
		return
	}

	var unlock utils.Unlocker
	if unlock, err = mutex.Lock(ctx); err != nil {
		return
	}
	defer func() {
		if ue := unlock(ctx); ue != nil {
			err = errors.CombineErrors(err, ue)
		}
	}()

	var data = map[string]string{}
	var ver int64

	switch ver, err = e.Get(ctx, key, &n); {
	case terrors.IsKeyNotExistsErr(err):
		data[key] = "1"
		if err = e.Create(ctx, data); err != nil {
			return
		}
		return 1, nil

	case err != nil:
		return
	}

	n++
	data[key] = strconv.FormatInt(int64(n), 10)
	err = e.Update(ctx, data, map[string]int64{key: ver})

	return
}

// Create .
func (e *ETCD) Create(ctx context.Context, data map[string]string, opts ...clientv3.OpOption) error {
	var ev = newTxnEvent()
	ev.data = data
	ev.opts = opts
	ev.txnErr = terrors.ErrKeyExists
	ev.vers = map[string]int64{}

	for k := range ev.data {
		ev.vers[k] = 0
	}

	return e.batchPut(ctx, ev)
}

// Update .
func (e *ETCD) Update(ctx context.Context, data map[string]string, vers map[string]int64, opts ...clientv3.OpOption) error {
	var ev = newTxnEvent()
	ev.data = data
	ev.opts = opts
	ev.txnErr = terrors.ErrKeyBadVersion
	ev.vers = vers

	return e.batchPut(ctx, ev)
}

func (e *ETCD) batchPut(ctx context.Context, ev *txnEvent) error {
	var ops, cmps = ev.generate()

	switch succ, err := e.BatchOperate(ctx, ops, cmps...); {
	case err != nil:
		return errors.Wrap(err, "")

	case !succ:
		return ev.txnErr
	}

	return nil
}

// Delete .
func (e *ETCD) Delete(ctx context.Context, keys []string, vers map[string]int64, opts ...clientv3.OpOption) error {
	var ev = newDelTxnEvent(keys, vers, opts...)
	var ops, cmps = ev.generate()

	switch succ, err := e.BatchOperate(ctx, ops, cmps...); {
	case err != nil:
		return errors.Wrap(err, "")

	case !succ:
		return errors.Wrap(terrors.ErrKeyBadVersion, "ETCD.Delete")
	}

	return nil
}

// BatchOperate .
func (e *ETCD) BatchOperate(ctx context.Context, ops []clientv3.Op, cmps ...clientv3.Cmp) (bool, error) {
	e.Lock()
	defer e.Unlock()

	var txn = e.cliv3.Txn(ctx)
	var resp, err = txn.If(cmps...).Then(ops...).Commit()
	if err != nil {
		return false, errors.Wrap(err, "")
	}

	return resp.Succeeded, nil
}

// GetPrefix .
func (e *ETCD) GetPrefix(ctx context.Context, prefix string, limit int64) (map[string][]byte, map[string]int64, error) {
	e.Lock()
	defer e.Unlock()

	var resp, err = e.cliv3.Get(ctx, prefix, clientv3.WithLimit(limit), clientv3.WithPrefix())
	switch {
	case err != nil:
		return nil, nil, errors.Wrap(err, "")
	case resp.Count < 1:
		return nil, nil, errors.Wrapf(terrors.ErrKeyNotExists, prefix)
	}

	var data = map[string][]byte{}
	var vers = map[string]int64{}

	for _, kv := range resp.Kvs {
		var key = string(kv.Key)
		data[key] = kv.Value
		vers[key] = kv.Version
	}

	return data, vers, nil
}

// Exists .
func (e *ETCD) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	var exists = map[string]bool{}

	for _, k := range keys {
		var resp, err = e.cliv3.Get(ctx, k, clientv3.WithKeysOnly())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		exists[k] = resp.Count > 0
	}

	return exists, nil
}

// Get .
func (e *ETCD) Get(ctx context.Context, key string, obj any, opts ...clientv3.OpOption) (int64, error) {
	e.Lock()
	defer e.Unlock()

	switch resp, err := e.cliv3.Get(ctx, key, opts...); {
	case err != nil:
		return 0, errors.Wrap(err, "")

	case resp.Count != 1:
		return 0, errors.Wrapf(terrors.ErrKeyNotExists, key)

	default:
		return resp.Kvs[0].Version, decode(resp.Kvs[0].Value, obj)
	}
}

// NewMutex .
func (e *ETCD) NewMutex(key string) (utils.Locker, error) {
	return NewMutex(e.cliv3.(*clientv3.Client), key)
}

// Close .
func (e *ETCD) Close() error {
	e.Lock()
	defer e.Unlock()
	return e.cliv3.Close()
}

// RetryTimedOut .
func RetryTimedOut(fn func() error, retryTimes int) error {
	for retried := 0; ; retried++ {
		if err := fn(); err != nil {
			if retried < retryTimes && terrors.IsETCDServerTimedOutErr(err) {
				log.Warnf(context.TODO(), "etcdserver: request timed out, retry it")
				continue
			}

			return err
		}

		return nil
	}
}
