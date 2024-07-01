package utils

import (
	"testing"

	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestSyncMap(t *testing.T) {
	mp := NewSyncMap()

	_, err := mp.Get("a", 0)
	assert.Equal(t, terrors.ErrKeyNotExists, err)

	mp.Put("a", 0, 5)
	v, err := mp.Get("a", 0)
	assert.NilErr(t, err)
	assert.Equal(t, 5, v)

	mp.Put("a", 0, 9)
	v, err = mp.Get("a", 0)
	assert.NilErr(t, err)
	assert.Equal(t, 9, v)
}
