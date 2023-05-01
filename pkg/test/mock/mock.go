package mock

import testify "github.com/stretchr/testify/mock"

// Anything .
const Anything = testify.Anything

// Mock .
type Mock = testify.Mock

// Ret .
type Ret struct {
	testify.Arguments
}

// NewRet .
func NewRet(args testify.Arguments) *Ret {
	return &Ret{args}
}

// Err .
func (r *Ret) Err(index int) (err error) {
	if obj := r.Get(index); obj != nil {
		err = obj.(error) //nolint
	}
	return
}

// Bytes .
func (r *Ret) Bytes(index int) (buf []byte) {
	if obj := r.Get(index); obj != nil {
		buf = obj.([]byte) //nolint
	}
	return
}
