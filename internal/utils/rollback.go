package utils

import (
	"context"
	"sync"
)

type RollbackFunc func() error

type rollbackListEntry struct {
	fn  RollbackFunc
	msg string
}

type RollbackList struct {
	mu   sync.Mutex
	List []rollbackListEntry
}

type ctxType string

const (
	rbKey ctxType = "rollbackList"
)

func NewRollbackListContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, rbKey, &RollbackList{})
}

func GetRollbackListFromContext(ctx context.Context) *RollbackList {
	v := ctx.Value(rbKey)
	if v != nil {
		return v.(*RollbackList)
	}
	return nil
}

func (rl *RollbackList) Append(fn RollbackFunc, msg string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.List = append(rl.List, rollbackListEntry{
		fn:  fn,
		msg: msg,
	})
}

func (rl *RollbackList) Pop() (fn RollbackFunc, msg string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	n := len(rl.List)
	if n > 0 {
		entry := rl.List[n-1]
		fn, msg = entry.fn, entry.msg
		rl.List = rl.List[:n-1]
	}
	return
}
