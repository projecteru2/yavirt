package utils

import (
	"os"
	"strconv"
)

const (
	hnEnv = "HOSTNAME"

	biggestMultiple1024 int64 = 0x7ffffffffffffc00
)

// Hostname .
func Hostname() (string, error) {
	if hn := os.Getenv(hnEnv); len(hn) > 0 {
		return hn, nil
	}
	return os.Hostname()
}

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
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max .
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MaxInt64 .
func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MinInt64 .
func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Atoi64 .
func Atoi64(s string) (int64, error) {
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
