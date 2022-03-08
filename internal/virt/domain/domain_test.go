package domain

import (
	"testing"

	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	libmocks "github.com/projecteru2/yavirt/pkg/libvirt/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func init() {
	model.Setup()
}

func TestSetSpec(t *testing.T) {
	libdom := &libmocks.Domain{}
	defer libdom.AssertExpectations(t)

	dom := newMockedDomain(t)
	dom.virt.(*libmocks.Libvirt).On("LookupDomain", mock.Anything).Return(libdom, nil).Once()
	defer func() { dom.virt.(*libmocks.Libvirt).AssertExpectations(t) }()

	libdom.On("Free").Return().Once()
	libdom.On("SetVcpusFlags", uint(1), libvirt.DomainVcpuConfig|libvirt.DomainVcpuMaximum).Return(nil).Once()
	libdom.On("SetVcpusFlags", uint(1), libvirt.DomainVcpuConfig|libvirt.DomainVcpuCurrent).Return(nil).Once()
	libdom.On("SetMemoryFlags", uint64(util.GB>>10), libvirt.DomainMemConfig|libvirt.DomainMemMaximum).Return(nil).Once()
	libdom.On("SetMemoryFlags", uint64(util.GB>>10), libvirt.DomainMemConfig|libvirt.DomainMemCurrent).Return(nil).Once()

	assert.NilErr(t, dom.SetSpec(1, util.GB))
}

func newMockedDomain(t *testing.T) *VirtDomain {
	gmod, err := model.NewGuest(nil, nil)
	assert.NilErr(t, err)

	return &VirtDomain{
		guest: gmod,
		virt:  &libmocks.Libvirt{},
	}
}
