package vmcache

import (
	"fmt"

	"github.com/digitalocean/go-libvirt"
)

func ToUint64(p libvirt.TypedParam) (val uint64, err error) {
	switch p.Value.D {
	case 1:
		val = uint64(p.Value.I.(int32))
	case 2:
		val = uint64(p.Value.I.(uint32))
	case 3:
		val = uint64(p.Value.I.(int64))
	case 4:
		val = p.Value.I.(uint64) //nolint
	case 6:
		val = uint64(p.Value.I.(int32))
	default:
		err = fmt.Errorf("invalid parameter type %v", p.Value.D)
	}
	return
}

func ToFloat64(p libvirt.TypedParam) (float64, error) {
	switch p.Value.D {
	case 5:
		return p.Value.I.(float64), nil
	default:
		return 0, fmt.Errorf("invalid parameter type %v", p.Value.D)
	}
}

func ToString(p libvirt.TypedParam) (string, error) {
	switch p.Value.D {
	case 7:
		return p.Value.I.(string), nil
	default:
		return "", fmt.Errorf("invalid parameter type %v", p.Value.D)
	}
}
