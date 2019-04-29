package util

import (
	"testing"

	"github.com/projecteru2/yavirt/test/assert"
)

func TestBitmap32WithZeroCount(t *testing.T) {
	bm := NewBitmap32(0)
	assert.Equal(t, 1, bm.Count)
	assert.Equal(t, 1, len(bm.Slices))
}

func TestBitmap32SingleSection(t *testing.T) {
	for count := 1; count <= bitsPerSection; count++ {
		bm := NewBitmap32(count)
		assert.Equal(t, count, bm.Count)
		assert.Equal(t, 1, len(bm.Slices), "expect only one slice for %d", count)
	}
}

func TestBitmap32MultipleSection(t *testing.T) {
	for count := bitsPerSection + 1; count < 2*bitsPerSection; count++ {
		bm := NewBitmap32(count)
		assert.Equal(t, count, bm.Count)
		assert.Equal(t, 2, len(bm.Slices), "expect 2 slices for %d", count)
	}
}

func TestBitmap32Bitwise(t *testing.T) {
	for count := 1; count <= bitsPerSection; count++ {
		bm := NewBitmap32(count)
		assert.Equal(t, []uint32{0}, bm.Slices)

		for i := 0; i < count; i++ {
			has, err := bm.Has(i)
			assert.Nil(t, err)
			assert.False(t, has)
		}

		has, err := bm.Has(count)
		assert.Err(t, err)
		assert.False(t, has)
		assert.True(t, bm.Available())

		for i := 0; i < count; i++ {
			assert.Nil(t, bm.Set(i))
			assert.True(t, SetBitFlags[i] <= bm.Slices[0])

			has, err := bm.Has(i)
			assert.Nil(t, err)
			assert.True(t, has)

			bm.Range(func(offset int, set bool) bool {
				if offset <= i {
					assert.True(t, set)
				} else {
					assert.False(t, set)
				}
				return true
			})
		}

		assert.False(t, bm.Available())

		for i := 0; i < count; i++ {
			assert.Nil(t, bm.Unset(i))

			has, err := bm.Has(i)
			assert.Nil(t, err)
			assert.False(t, has)
			assert.True(t, bm.Available())

			bm.Range(func(offset int, set bool) bool {
				if offset <= i {
					assert.False(t, set)
				} else {
					assert.True(t, set)
				}
				return true
			})
		}

		assert.True(t, bm.Available())
	}
}
