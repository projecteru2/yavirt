package util

import (
	"strconv"
	"sync"

	"github.com/projecteru2/yavirt/pkg/errors"
)

type ExitCodeMap struct {
	sync.Map
}

func NewSyncMap() *ExitCodeMap {
	return &ExitCodeMap{}
}

// Get .
func (s *ExitCodeMap) Get(id string, pid int) (int, error) {
	v, ok := s.Load(id + strconv.Itoa(pid))
	if !ok {
		return 0, errors.ErrKeyNotExists
	}
	return v.(int), nil
}

// Put .
func (s *ExitCodeMap) Put(id string, pid int, value int) {
	s.Store(id+strconv.Itoa(pid), value)
}
