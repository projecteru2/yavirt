package libvirt

import libvirtgo "github.com/libvirt/libvirt-go"

// IsErrNoDomain is the err indicating not exists.
func IsErrNoDomain(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(libvirtgo.Error); ok {
		return e.Code == libvirtgo.ERR_NO_DOMAIN
	}

	return false
}
