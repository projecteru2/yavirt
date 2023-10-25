package libvirt

import (
	"net"
	"time"

	"github.com/projecteru2/yavirt/pkg/errors"
	golibvirt "github.com/projecteru2/yavirt/third_party/libvirt"
	"github.com/projecteru2/yavirt/third_party/libvirt/socket/dialers"
)

// Libvirt uses to interact with libvirtd service.
type Libvirt interface {
	Close() (int, error)
	LookupDomain(string) (Domain, error)
	DefineDomain(string) (Domain, error)
	ListDomainsNames() ([]string, error)
	GetAllDomainStats(doms []golibvirt.Domain) ([]golibvirt.DomainStatsRecord, error)
}

// Libvirtee is a Libvirt implement.
type Libvirtee struct {
	*golibvirt.Libvirt
}

func (l *Libvirtee) Close() (int, error) {
	err := l.ConnectClose()
	if err != nil {
		return 0, err
	}
	return 1, nil
}

// Connect connects a guest's domain.
func Connect(uri string) (l *Libvirtee, err error) {
	c, err := net.DialTimeout("unix", "/var/run/libvirt/libvirt-sock", 5*time.Second)
	if err != nil {
		return nil, err
	}
	l = &Libvirtee{}
	l.Libvirt = golibvirt.NewWithDialer(dialers.NewAlreadyConnected(c))
	if err = l.ConnectToURI(golibvirt.ConnectURI(uri)); err != nil {
		return nil, err
	}
	return
}

// DefineDomain defines a new domain.
func (l *Libvirtee) DefineDomain(xml string) (Domain, error) {
	raw, err := l.DomainDefineXML(xml)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return NewDomainee(l.Libvirt, &raw), nil
}

// LookupDomain looks up a domain by name.
func (l *Libvirtee) LookupDomain(name string) (Domain, error) {
	raw, err := l.DomainLookupByName(name)
	if err != nil {
		if IsErrNoDomain(err) {
			return nil, errors.Annotatef(errors.ErrDomainNotExists, name)
		}
		return nil, errors.Trace(err)
	}
	return NewDomainee(l.Libvirt, &raw), nil
}

// ListDomainsNames lists all domains' name.
func (l *Libvirtee) ListDomainsNames() ([]string, error) {
	raw, err := l.ListAllDomains()
	if err != nil {
		return nil, errors.Trace(err)
	}

	names := make([]string, len(raw))
	for i, d := range raw {
		names[i] = d.Name
	}

	return names, nil
}

// ListAllDomains lists all domains regardless the state.
func (l *Libvirtee) ListAllDomains() ([]golibvirt.Domain, error) {
	flags := golibvirt.ConnectListDomainsActive | golibvirt.ConnectListDomainsInactive
	dList, _, err := l.ConnectListAllDomains(int32(flags), ListAllDomainFlags)
	return dList, err
}

func (l *Libvirtee) GetAllDomainStats(doms []golibvirt.Domain) ([]golibvirt.DomainStatsRecord, error) {
	flags := golibvirt.ConnectGetAllDomainsStatsRunning
	var statsType golibvirt.DomainStatsTypes
	return l.ConnectGetAllDomainStats(doms, uint32(statsType), flags)
}
