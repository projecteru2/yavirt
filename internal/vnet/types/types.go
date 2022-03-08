package types

import (
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// EndpointArgs .
type EndpointArgs struct {
	EndpointID      string
	IPs             []meta.IP
	Device          device.VirtLink
	MAC             string
	Hostname        string
	ResourceVersion string
	UID             string
	Profiles        []string
}

// Check .
func (a EndpointArgs) Check() error {
	switch {
	case len(a.EndpointID) < 1:
		return errors.Annotatef(errors.ErrInvalidValue, "EndpointID is empty")

	case len(a.IPs) < 1:
		return errors.Annotatef(errors.ErrInvalidValue, "IPs is empty")

	case a.Device == nil:
		return errors.Annotatef(errors.ErrInvalidValue, "Device is nil")

	case len(a.MAC) < 1:
		return errors.Annotatef(errors.ErrInvalidValue, "MAC is empty")

	case len(a.Hostname) < 1:
		return errors.Annotatef(errors.ErrInvalidValue, "Hostname is empty")

	default:
		return nil
	}
}
