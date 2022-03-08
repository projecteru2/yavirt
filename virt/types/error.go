package types

import (
	"fmt"

	"github.com/projecteru2/yavirt/pkg/libvirt"
)

// DomainStateErr .
type DomainStateErr struct {
	Exps []libvirt.DomainState
	Act  libvirt.DomainState
}

// NewDomainStatesErr .
func NewDomainStatesErr(act libvirt.DomainState, exps ...libvirt.DomainState) *DomainStateErr {
	return &DomainStateErr{
		Exps: exps,
		Act:  act,
	}
}

func (e *DomainStateErr) Error() string {
	return fmt.Sprintf("require %v, but guest is %s",
		libvirt.GetDomainStatesStrings(e.Exps),
		libvirt.GetDomainStateString(e.Act))
}
