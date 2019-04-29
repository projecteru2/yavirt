package util

import (
	"crypto/rand"
	"math/big"
	"time"
)

// RandAfterFunc .
func RandAfterFunc(min, max time.Duration, fn func()) *time.Timer {
	t, _ := rand.Int(rand.Reader, big.NewInt(int64(max)-int64(min)))
	dur := time.Duration(t.Int64()) + min
	return time.AfterFunc(dur, fn)
}
