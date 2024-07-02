package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRollbacklList(t *testing.T) {
	ctx := NewRollbackListContext(context.Background())
	rl := GetRollbackListFromContext(ctx)
	rl.Append(func() error { return nil }, "1")
	rl.Append(func() error { return nil }, "2")
	rl2 := GetRollbackListFromContext(ctx)
	fn, msg := rl2.Pop()
	assert.Equal(t, msg, "2")
	assert.Nil(t, fn())
	fn, msg = rl2.Pop()
	assert.Equal(t, msg, "1")
	assert.Nil(t, fn())
}
