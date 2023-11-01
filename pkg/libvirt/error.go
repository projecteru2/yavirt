package libvirt

import libvirtgo "github.com/projecteru2/yavirt/third_party/libvirt"

// IsErrNoDomain is the err indicating not exists.
func IsErrNoDomain(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(libvirtgo.Error); ok {
		return e.Code == uint32(libvirtgo.ErrNoDomain)
	}

	return false
}
