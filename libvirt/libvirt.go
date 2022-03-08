package libvirt

import (
	libvirtgo "github.com/libvirt/libvirt-go"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Libvirt uses to interact with libvirtd service.
type Libvirt interface {
	Close() (int, error)
	LookupDomain(string) (Domain, error)
	DefineDomain(string) (Domain, error)
	ListDomainsNames() ([]string, error)
}

// Libvirtee is a Libvirt implement.
type Libvirtee struct {
	*libvirtgo.Connect
}

// Connect connects a guest's domain.
func Connect(uri string) (l *Libvirtee, err error) {
	l = &Libvirtee{}
	l.Connect, err = libvirtgo.NewConnect(uri)
	return
}

// ListAllDomains lists all domains regardless the state.
func (l *Libvirtee) ListAllDomains() ([]libvirtgo.Domain, error) {
	return l.Connect.ListAllDomains(ListAllDomainFlags)
}

// DefineDomain defines a new domain.
func (l *Libvirtee) DefineDomain(xml string) (Domain, error) {
	raw, err := l.Connect.DomainDefineXML(xml)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return NewDomainee(raw), nil
}

// LookupDomain looks up a domain by name.
func (l *Libvirtee) LookupDomain(name string) (Domain, error) {
	raw, err := l.Connect.LookupDomainByName(name)
	if err != nil {
		if IsErrNoDomain(err) {
			return nil, errors.Annotatef(errors.ErrDomainNotExists, name)
		}
		return nil, errors.Trace(err)
	}
	return NewDomainee(raw), nil
}

// ListDomainsNames lists all domains' name.
func (l *Libvirtee) ListDomainsNames() ([]string, error) {
	raw, err := l.Connect.ListAllDomains(ListAllDomainFlags)
	if err != nil {
		return nil, errors.Trace(err)
	}

	names := make([]string, len(raw))
	for i, d := range raw {
		if names[i], err = d.GetName(); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return names, nil
}
