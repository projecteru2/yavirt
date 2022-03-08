package utils

import (
	"github.com/projecteru2/yavirt/pkg/errors"
)

const bitsPerSection = 32

var (
	// SetBitFlags .
	SetBitFlags [bitsPerSection]uint32
	// UnsetBitFlags .
	UnsetBitFlags [bitsPerSection]uint32
)

func init() { //nolint
	for i := 0; i < bitsPerSection; i++ {
		SetBitFlags[i] = 1 << uint(i)
		UnsetBitFlags[i] = ^SetBitFlags[i]
	}
}

// Bitmap32 .
type Bitmap32 struct {
	Slices []uint32 `json:"slices"`
	Count  int      `json:"total_count"`
}

// NewBitmap32 .
func NewBitmap32(count int) *Bitmap32 {
	count = Max(1, count)

	leng := count / bitsPerSection
	if count%bitsPerSection > 0 {
		leng++
	}

	return &Bitmap32{
		Slices: make([]uint32, leng),
		Count:  count,
	}
}

// Unset .
func (b *Bitmap32) Unset(offset int) (err error) {
	if err = b.checkOffset(offset); err != nil {
		return
	}

	mi, bi := b.GetIndex(offset)

	b.Slices[mi] &= UnsetBitFlags[bi]

	return
}

// Set .
func (b *Bitmap32) Set(offset int) (err error) {
	if err = b.checkOffset(offset); err != nil {
		return
	}

	mi, bi := b.GetIndex(offset)

	b.Slices[mi] |= SetBitFlags[bi]

	return
}

// Has .
func (b *Bitmap32) Has(offset int) (has bool, err error) {
	if err = b.checkOffset(offset); err != nil {
		return
	}

	mi, bi := b.GetIndex(offset)

	has = b.has(mi, bi)

	return
}

// Available .
func (b *Bitmap32) Available() bool {
	allOnes := (1 << uint(b.bitsLen())) - 1

	for _, sec := range b.Slices {
		if sec < uint32(allOnes) {
			return true
		}
	}

	return false
}

// Range .
func (b *Bitmap32) Range(f func(offset int, set bool) bool) {
	for mi := 0; mi < len(b.Slices); mi++ {

		for bi := 0; bi < b.bitsLen(); bi++ {

			offset := b.getOffset(mi, bi)
			set := b.has(mi, bi)

			if !f(offset, set) {
				return
			}
		}
	}
}

func (b *Bitmap32) getOffset(mi, bi int) int {
	return mi*b.bitsLen() + bi
}

func (b *Bitmap32) has(mi, bi int) bool {
	return b.Slices[mi]&SetBitFlags[bi] > 0
}

func (b *Bitmap32) bitsLen() int {
	return Min(bitsPerSection, b.Count)
}

// GetIndex .
func (b *Bitmap32) GetIndex(offset int) (mi, bi int) {
	mi = offset / bitsPerSection
	bi = offset % bitsPerSection
	return
}

func (b *Bitmap32) checkOffset(offset int) error {
	if offset >= b.Count {
		return errors.Annotatef(errors.ErrTooLargeOffset,
			"at most %d, but %d", b.Count-1, offset)
	}
	return nil
}
