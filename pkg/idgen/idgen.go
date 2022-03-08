package idgen

import (
	"fmt"
	"strconv"
	"time"

	"github.com/projecteru2/yavirt/pkg/utils"
)

// Generator .
type Generator struct {
	prefix  uint32
	suffix  uint64
	counter utils.AtomicInt64
}

// New .
func New(memberID uint32, seed time.Time) *Generator {
	return &Generator{
		prefix: memberID,
		suffix: uint64(seed.UnixNano()) / uint64(time.Microsecond),
	}
}

// Next .
func (g *Generator) Next() string {
	var b36 = strconv.FormatInt(g.counter.Incr(), 36) //nolint:gomnd // because its base36
	return fmt.Sprintf("%010d%017d%05s", g.prefix, g.suffix, b36)
}

var gen *Generator

// Setup .
func Setup(memberID uint32, seed time.Time) {
	gen = New(memberID, seed)
}

// Next .
func Next() string {
	return gen.Next()
}
