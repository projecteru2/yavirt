package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"time"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func VirtID(id string) string {
	req := types.GuestReq{ID: id}
	return req.VirtID()
}

func NewCreateSessionFlock(id string) *utils.Flock {
	var fn = fmt.Sprintf("guest_create_session_%s.flock", id)
	var fpth = filepath.Join(configs.Conf.VirtFlockDir, fn)
	return utils.NewFlock(fpth)
}

// EnsureReaderClosed As the name says,
// blocks until the stream is empty, until we meet EOF
func EnsureReaderClosed(stream io.ReadCloser) {
	if stream == nil {
		return
	}
	if _, err := io.Copy(io.Discard, stream); err != nil {
		log.Errorf(context.TODO(), err, "Empty stream failed")
	}
	_ = stream.Close()
}

func RandomString(length int) string {
	b := make([]byte, length+2)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

// WithTimeout runs a function with given timeout
func WithTimeout(ctx context.Context, timeout time.Duration, f func(ctx2 context.Context)) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	f(ctx)
}

// GetMaxAttemptsByTTL .
func GetMaxAttemptsByTTL(ttl int64) int {
	// if selfmon is enabled, retry 5 times
	if ttl < 1 {
		return 5
	}
	return int(math.Floor(math.Log2(float64(ttl)+1))) + 1
}
