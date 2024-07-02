package types

import (
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

type CalicoArgs struct {
	IPPool    string `json:"ippool"`
	Namespace string `json:"namespace"`

	ResourceVersion string
	UID             string
	Profiles        []string
}

type OVNArgs struct {
	LogicalSwitchUUID     string `json:"logical_switch_uuid"`
	LogicalSwitchName     string `json:"logical_switch_name"`
	LogicalSwitchPortName string `json:"logical_switch_port_name"`
}

type VlanArgs struct {
	// TODO
}

type CNIArgs struct {
}

// EndpointArgs .
type EndpointArgs struct {
	GuestID    string
	EndpointID string
	IPs        []meta.IP
	DevName    string
	MAC        string
	MTU        int
	Hostname   string

	Calico CalicoArgs
	OVN    OVNArgs
	Vlan   VlanArgs
	CNI    CNIArgs
}

// Check .
func (a EndpointArgs) Check() error {
	switch {
	case len(a.EndpointID) < 1:
		return errors.Wrapf(terrors.ErrInvalidValue, "EndpointID is empty")

	case len(a.IPs) < 1:
		return errors.Wrapf(terrors.ErrInvalidValue, "IPs is empty")

	case a.DevName == "":
		return errors.Wrapf(terrors.ErrInvalidValue, "Device is nil")

	case len(a.MAC) < 1:
		return errors.Wrapf(terrors.ErrInvalidValue, "MAC is empty")

	case len(a.Hostname) < 1:
		return errors.Wrapf(terrors.ErrInvalidValue, "Hostname is empty")

	default:
		return nil
	}
}
