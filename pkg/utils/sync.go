package utils

import (
	"context"
	"sync/atomic"
)

// Locker .
type Locker interface {
	Lock(context.Context) (Unlocker, error)
}

// Unlocker func.
type Unlocker func(context.Context) error

// Once .
type Once struct {
	done int32
}

// Do .
func (o *Once) Do(fn func() error) (err error) {
	if !atomic.CompareAndSwapInt32(&o.done, 0, 1) {
		return
	}

	if err = fn(); err != nil {
		// rollback
		atomic.StoreInt32(&o.done, 0)
	}

	return
}

// AtomicInt64 .
type AtomicInt64 struct {
	i64 int64
}

// Int64 .
func (a *AtomicInt64) Int64() int64 {
	return atomic.LoadInt64(&a.i64)
}

// Incr .
func (a *AtomicInt64) Incr() int64 {
	return a.Add(1)
}

// Add .
func (a *AtomicInt64) Add(v int64) int64 {
	return atomic.AddInt64(&a.i64, v)
}

// Set .
func (a *AtomicInt64) Set(v int64) {
	atomic.StoreInt64(&a.i64, v)
}
