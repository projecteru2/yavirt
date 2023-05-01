package store

import (
	"context"

	"go.etcd.io/etcd/clientv3"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Store .
type Store interface {
	Create(ctx context.Context, data map[string]string, opts ...clientv3.OpOption) error

	Get(ctx context.Context, key string, obj any, opts ...clientv3.OpOption) (ver int64, err error)
	GetPrefix(ctx context.Context, prefix string, limit int64) (data map[string][]byte, vers map[string]int64, err error)
	Exists(ctx context.Context, keys []string) (exists map[string]bool, err error)

	Update(ctx context.Context, data map[string]string, vers map[string]int64, opts ...clientv3.OpOption) error
	BatchOperate(ctx context.Context, ops []clientv3.Op, cmps ...clientv3.Cmp) (succeeded bool, err error)
	Delete(ctx context.Context, keys []string, vers map[string]int64, opts ...clientv3.OpOption) error

	Close() error

	IncrUint32(ctx context.Context, key string) (uint32, error)
	NewMutex(key string) (utils.Locker, error)
}

// New .
func New(metatype string) (Store, error) {
	switch metatype {
	case "etcd":
		return etcd.New()
	default:
		return nil, errors.Annotatef(errors.ErrInvalidValue, "invalid meta type: %s", metatype)
	}
}

var store Store

// Setup .
func Setup(metatype string) (err error) {
	store, err = New(metatype)
	return
}

// SetStore .
func SetStore(s Store) {
	store = s
}

// GetStore .
func GetStore() Store {
	return store
}

// Create .
func Create(ctx context.Context, data map[string]string, opts ...clientv3.OpOption) error {
	return store.Create(ctx, data, opts...)
}

// Update .
func Update(ctx context.Context, data map[string]string, vers map[string]int64, opts ...clientv3.OpOption) error {
	return store.Update(ctx, data, vers, opts...)
}

// BatchOperate .
func BatchOperate(ctx context.Context, ops []clientv3.Op, cmps ...clientv3.Cmp) (bool, error) {
	return store.BatchOperate(ctx, ops, cmps...)
}

// Get .
func Get(ctx context.Context, key string, obj any, opts ...clientv3.OpOption) (int64, error) {
	return store.Get(ctx, key, obj, opts...)
}

// Exists .
func Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	return store.Exists(ctx, keys)
}

// GetPrefix .
func GetPrefix(ctx context.Context, prefix string, limit int64) (map[string][]byte, map[string]int64, error) {
	return store.GetPrefix(ctx, prefix, limit)
}

// Delete .
func Delete(ctx context.Context, keys []string, vers map[string]int64, opts ...clientv3.OpOption) error {
	return store.Delete(ctx, keys, vers, opts...)
}

// Close .
func Close() error {
	if store != nil {
		return store.Close()
	}
	return nil
}

// IncrUint32 .
func IncrUint32(ctx context.Context, key string) (uint32, error) {
	return store.IncrUint32(ctx, key)
}

// Lock .
func Lock(ctx context.Context, key string) (utils.Unlocker, error) {
	mutex, err := store.NewMutex(key)
	if err != nil {
		return nil, errors.Annotatef(err, "create mutex %s failed", key)
	}
	return mutex.Lock(ctx)
}
