package domain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	libmocks "github.com/projecteru2/yavirt/pkg/libvirt/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestSetSpec(t *testing.T) {
	libdom := &libmocks.Domain{}
	defer libdom.AssertExpectations(t)

	dom := newMockedDomain(t)
	dom.virt.(*libmocks.Libvirt).On("LookupDomain", mock.Anything).Return(libdom, nil).Once()
	defer func() { dom.virt.(*libmocks.Libvirt).AssertExpectations(t) }()

	libdom.On("SetVcpusFlags", uint(1), libvirt.DomainVcpuConfig|libvirt.DomainVcpuMaximum).Return(nil).Once()
	libdom.On("SetVcpusFlags", uint(1), libvirt.DomainVcpuConfig|libvirt.DomainVcpuCurrent).Return(nil).Once()
	libdom.On("SetMemoryFlags", uint64(utils.GB>>10), libvirt.DomainMemConfig|libvirt.DomainMemMaximum).Return(nil).Once()
	libdom.On("SetMemoryFlags", uint64(utils.GB>>10), libvirt.DomainMemConfig|libvirt.DomainMemCurrent).Return(nil).Once()

	assert.NilErr(t, dom.SetSpec(1, utils.GB))
}

// func TestAttachGPU(t *testing.T) {
// 	libdom := &libmocks.Domain{}
// 	defer libdom.AssertExpectations(t)

// 	dom := newMockedDomain(t)
// 	dom.virt.(*libmocks.Libvirt).On("LookupDomain", mock.Anything).Return(libdom, nil).Once()
// 	defer func() { dom.virt.(*libmocks.Libvirt).AssertExpectations(t) }()
// 	libdom.On("GetXMLDesc", mock.Anything).Return("", nil).Once()
// }

func TestExtractHostdevXML(t *testing.T) {
	x := `
<domain type='kvm'>
  <name>haha</name>
  <uuid>bbb</uuid>
  <sysinfo type='smbios'>
    <bios>
      <entry name='vendor'>YAVIRT</entry>
    </bios>
  </sysinfo>
  <os>
    <type arch='x86_64'>hvm</type>
    <smbios mode='sysinfo'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>restart</on_crash>
  <pm>
    <suspend-to-mem enabled='no'/>
    <suspend-to-disk enabled='no'/>
  </pm>
  <devices>
    <hostdev mode='subsystem' type='pci' managed='yes'>
      <source>
      <address domain='0x0000' bus='0x81' slot='0x00' function='0x0'/>
      </source>
    </hostdev>
    <hostdev mode='subsystem' type='pci' managed='yes'>
      <source>
      <address domain='0x0000' bus='0x82' slot='0x00' function='0x0'/>
      </source>
    </hostdev>
  </devices>
</domain>

	`
	doc, err := xmlquery.Parse(strings.NewReader(x))
	assert.Nil(t, err)
	xml, err := extractHostdevXML(doc, "0000:81:00.0")
	assert.Nil(t, err)
	assert.Equal(t, `<hostdev mode="subsystem" type="pci" managed="yes"><source><address domain="0x0000" bus="0x81" slot="0x00" function="0x0"></address></source></hostdev>`, xml, "xml is incorrect")
	fmt.Printf("%s\n", xml)
}
func newMockedDomain(t *testing.T) *VirtDomain {
	gmod, err := models.NewGuest(nil, nil)
	assert.NilErr(t, err)

	return &VirtDomain{
		guest: gmod,
		virt:  &libmocks.Libvirt{},
	}
}
