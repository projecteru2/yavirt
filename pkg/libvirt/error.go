package libvirt

import golibvirt "github.com/projecteru2/yavirt/third_party/libvirt"

// IsErrNoDomain is the err indicating not exists.
func IsErrNoDomain(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(golibvirt.Error); ok {
		return e.Code == uint32(golibvirt.ErrNoDomain)
	}

	return false
}
