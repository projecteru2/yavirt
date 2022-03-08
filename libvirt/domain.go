package libvirt

import (
	libvirtgo "github.com/libvirt/libvirt-go"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// Domain .
type Domain interface {
	Create() error
	ShutdownFlags(flags DomainShutdownFlags) error
	Destroy() error
	DestroyFlags(flags DomainDestroyFlags) error
	UndefineFlags(flags DomainUndefineFlags) error
	Suspend() error
	Resume() error

	SetVcpusFlags(vcpu uint, flags DomainVcpuFlags) error
	SetMemoryFlags(memory uint64, flags DomainMemoryModFlags) error
	AmplifyVolume(filepath string, cap uint64) error
	AttachVolume(xml string) (DomainState, error)

	Free()

	GetState() (DomainState, error)
	GetInfo() (*DomainInfo, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags DomainXMLFlags) (string, error)
	GetName() (string, error)
}

// Domainee is a implement of Domain.
type Domainee struct {
	*libvirtgo.Domain
}

// NewDomainee converts a libvirt-go Domain object to a *Domainee object.
func NewDomainee(raw *libvirtgo.Domain) (dom *Domainee) {
	dom = &Domainee{}
	dom.Domain = raw
	return
}

// Free .
func (d *Domainee) Free() {
	if err := d.Domain.Free(); err != nil {
		log.ErrorStack(err)
	}
}

// AttachVolume .
func (d *Domainee) AttachVolume(xml string) (st DomainState, err error) {
	flags := DomainDeviceModifyConfig | DomainDeviceModifyCurrent

	switch st, err = d.GetState(); {
	case err != nil:
		return
	case st == DomainRunning:
		flags |= DomainDeviceModifyLive
	case st != DomainShutoff:
		return DomainNoState, errors.Annotatef(errors.ErrInvalidValue, "invalid domain state: %v", st)
	}

	err = d.Domain.AttachDeviceFlags(xml, flags)

	return
}

// GetState .
func (d *Domainee) GetState() (st DomainState, err error) {
	st, _, err = d.Domain.GetState()
	return
}

// AmplifyVolume .
func (d *Domainee) AmplifyVolume(filepath string, cap uint64) error {
	return d.Domain.BlockResize(filepath, cap, DomainBlockResizeBytes)
}
