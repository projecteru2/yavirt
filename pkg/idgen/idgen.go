package idgen

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
	"time"
)

// Generator .
type Generator struct {
	memberID   uint32
	randPrefix uint32
	counter    uint32
}

// New .
func New(memberID uint32) (*Generator, error) {
	if memberID >= 100000 {
		return nil, fmt.Errorf("member id is too large: %d", memberID)
	}
	return &Generator{
		memberID:   memberID,
		randPrefix: getRandomUint32() % 1000000,
		counter:    getRandomUint32(),
	}, nil
}

// Next .
func (g *Generator) Next() string {
	counter := atomic.AddUint32(&g.counter, 1)
	var b36 = strconv.FormatInt(int64(counter), 36)
	suffix := uint64(time.Now().UnixMilli())
	return fmt.Sprintf("%05d%06d%013d%08s", g.memberID, g.randPrefix, suffix, b36)
}

var gen *Generator

// Setup .
func Setup(memberID uint32) (err error) {
	gen, err = New(memberID)
	return
}

// Next .
func Next() string {
	return gen.Next()
}

func CheckID(id string) bool {
	return len(id) >= 32
}

func getRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %w", err))
	}
	var ans uint32
	if err := binary.Read(bytes.NewBuffer(b[:]), binary.LittleEndian, &ans); err != nil {
		panic(fmt.Errorf("failed to convert byte array to integer %s: %s", b, err))
	}
	return ans
}
