package utils

import (
	"strconv"

	"github.com/projecteru2/core/utils"
	"golang.org/x/exp/constraints"
)

const (
	biggestMultiple1024 int64 = 0x7ffffffffffffc00
)

// Invoke .
func Invoke(funcs []func() error) error {
	for _, fn := range funcs {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// Min .
func Min[T constraints.Ordered](a, b T) T {
	return utils.Min(a, b)
}

// Max .
func Max[T constraints.Ordered](a, b T) T {
	return utils.Max(a, b)
}

// Atoi64 .
func Atoi64(s string) (int64, error) {
	// obvious base 10 and 64-bit number
	return strconv.ParseInt(s, 10, 64)
}

// NormalizeMultiple1024 the size must be multiple of 1024.
func NormalizeMultiple1024(size int64) int64 {
	switch {
	case size <= 0:
		return size
	case size >= biggestMultiple1024:
		return biggestMultiple1024
	default:
		return size & biggestMultiple1024
	}
}
