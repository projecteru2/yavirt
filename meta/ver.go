package meta

import "github.com/projecteru2/yavirt/util"

// Ver .
type Ver struct {
	ver util.AtomicInt64
}

// NewVer .
func NewVer() *Ver {
	return &Ver{}
}

// SetVer .
func (v *Ver) SetVer(ver int64) {
	v.ver.Set(ver)
}

// IncrVer .
func (v *Ver) IncrVer() {
	v.ver.Incr()
}

// GetVer .
func (v *Ver) GetVer() int64 {
	return v.ver.Int64()
}
