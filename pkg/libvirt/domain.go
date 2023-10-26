package libvirt

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	libvirtgo "github.com/projecteru2/yavirt/third_party/libvirt"
)

// Domain .
type Domain interface { //nolint
	Create() error
	SetAutostart(autostart bool) error
	ShutdownFlags(flags DomainShutdownFlags) error
	Destroy() error
	DestroyFlags(flags DomainDestroyFlags) error
	UndefineFlags(flags DomainUndefineFlags) error
	Suspend() error
	Resume() error

	SetVcpusFlags(vcpu uint, flags DomainVcpuFlags) error
	SetMemoryFlags(memory uint64, flags DomainMemoryModFlags) error
	SetMemoryStatsPeriod(period int, config, live bool) error
	AmplifyVolume(filepath string, cap uint64) error
	AttachVolume(xml string) (DomainState, error)
	DetachVolume(xml string) (st DomainState, err error)

	Free()

	GetState() (DomainState, error)
	GetInfo() (*DomainInfo, error)
	GetUUIDString() (string, error)
	GetXMLDesc(flags DomainXMLFlags) (string, error)
	GetName() (string, error)
	QemuAgentCommand(ctx context.Context, cmd string) (string, error)
	OpenConsole(devname string, flags *ConsoleFlags) (*Console, error)
}

// Domainee is a implement of Domain.
type Domainee struct {
	Libvirt *libvirtgo.Libvirt
	*libvirtgo.Domain
}

func (d *Domainee) Create() error {
	err := d.Libvirt.DomainCreate(*d.Domain)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) Free() {

}

func (d *Domainee) SetAutostart(autostart bool) error {
	autostartFlag := int32(0)
	if autostart {
		autostartFlag = 1
	}
	err := d.Libvirt.DomainSetAutostart(*d.Domain, autostartFlag)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) ShutdownFlags(flags DomainShutdownFlags) error {
	err := d.Libvirt.DomainShutdownFlags(*d.Domain, libvirtgo.DomainShutdownFlagValues(flags))
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) Destroy() error {
	err := d.Libvirt.DomainDestroy(*d.Domain)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) DestroyFlags(flags DomainDestroyFlags) error {
	err := d.Libvirt.DomainDestroyFlags(*d.Domain, flags)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) UndefineFlags(flags DomainUndefineFlags) error {
	err := d.Libvirt.DomainUndefineFlags(*d.Domain, flags)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) Suspend() error {
	err := d.Libvirt.DomainSuspend(*d.Domain)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) Resume() error {
	err := d.Libvirt.DomainResume(*d.Domain)
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) SetVcpusFlags(vcpu uint, flags DomainVcpuFlags) error {
	err := d.Libvirt.DomainSetVcpusFlags(*d.Domain, uint32(vcpu), uint32(flags))
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) SetMemoryFlags(memory uint64, flags DomainMemoryModFlags) error {
	err := d.Libvirt.DomainSetMemoryFlags(*d.Domain, memory, uint32(flags))
	if err != nil {
		return err
	}
	return nil
}

func (d *Domainee) GetInfo() (*DomainInfo, error) {
	rState, rMaxMem, rMemory, rNrVirtCPU, rCPUTime, err := d.Libvirt.DomainGetInfo(*d.Domain)
	if err != nil {
		return nil, err
	}
	info := &libvirtgo.DomainGetInfoRet{
		State:     rState,
		MaxMem:    rMaxMem,
		Memory:    rMemory,
		NrVirtCPU: rNrVirtCPU,
		CPUTime:   rCPUTime,
	}
	return info, nil
}

func (d *Domainee) GetUUIDString() (string, error) {
	return hex.EncodeToString(d.Domain.UUID[:]), nil
}

func (d *Domainee) GetXMLDesc(flags DomainXMLFlags) (string, error) {
	desc, err := d.Libvirt.DomainGetXMLDesc(*d.Domain, flags)
	if err != nil {
		return "", err
	}
	return desc, nil
}

func (d *Domainee) GetName() (string, error) {
	return d.Domain.Name, nil
}

// NewDomainee converts a libvirt-go Domain object to a *Domainee object.
func NewDomainee(lib *libvirtgo.Libvirt, raw *libvirtgo.Domain) (dom *Domainee) {
	dom = &Domainee{}
	dom.Libvirt = lib
	dom.Domain = raw
	return
}

func (d *Domainee) SetMemoryStatsPeriod(period int, config, live bool) error {
	flags := libvirtgo.DomainMemCurrent
	if config {
		flags |= libvirtgo.DomainMemConfig
	}
	if live {
		flags |= libvirtgo.DomainMemLive
	}
	return d.Libvirt.DomainSetMemoryStatsPeriod(*d.Domain, int32(period), flags)
}

func (d *Domainee) QemuAgentCommand(ctx context.Context, cmd string) (string, error) {
	flags := uint32(0)
	timeout := libvirtgo.DomainAgentResponseTimeoutDefault
	if deadline, ok := ctx.Deadline(); ok {
		remain := time.Until(deadline)
		timeout = libvirtgo.DomainAgentResponseTimeoutValues(remain.Seconds())
	}
	retStrArr, err := d.Libvirt.QEMUDomainAgentCommand(*d.Domain, cmd, int32(timeout), flags)
	if err != nil {
		log.Debugf("[Domainee_QemuAgentCommand] Libvirt.QEMUDomainAgentCommand err, msg: %s, trace: %s", err.Error(), errors.Trace(err).Error())
		return "", err
	}
	if len(retStrArr) == 0 {
		return "", nil
	}
	return retStrArr[0], nil
}

func (d *Domainee) OpenConsole(devname string, cf *ConsoleFlags) (*Console, error) {
	// 创建一个写入控制台输出的缓冲区
	st := NewStream()

	go func() {
		err := d.Libvirt.OpenConsole(*d.Domain, libvirtgo.OptString{devname}, st.GetInReader(), st.GetOutWriter(), uint32(cf.genLibvirtFlags()))
		if err != nil {
			log.Errorf("[Domainee:OpenConsole] Libvirt.DomainOpenConsole err: ", err.Error())
			return
		}
	}()

	// 打开虚拟机的控制台连接
	con := newConsole(st)
	if err := con.AddReadWriter(); err != nil {
		con.Close()
		return nil, err
	}

	return con, nil
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

	err = d.Libvirt.DomainAttachDeviceFlags(*d.Domain, xml, uint32(flags))

	return
}

// AttachVolume .
func (d *Domainee) DetachVolume(xml string) (st DomainState, err error) {
	flags := DomainDeviceModifyConfig | DomainDeviceModifyCurrent

	switch st, err = d.GetState(); {
	case err != nil:
		return
	case st == DomainRunning:
		flags |= DomainDeviceModifyLive
	case st != DomainShutoff:
		return DomainNoState, errors.Annotatef(errors.ErrInvalidValue, "invalid domain state: %v", st)
	}

	err = d.Libvirt.DomainDetachDeviceFlags(*d.Domain, xml, uint32(flags))

	return
}

// GetState .
func (d *Domainee) GetState() (st DomainState, err error) {
	flags := DomainNoState
	//flags := DomainDeviceModifyConfig | DomainDeviceModifyCurrent
	iSt, _, err := d.Libvirt.DomainGetState(*d.Domain, uint32(flags))
	st = DomainState(iSt)
	return
}

// AmplifyVolume .
func (d *Domainee) AmplifyVolume(filepath string, cap uint64) error {
	return d.Libvirt.DomainBlockResize(*d.Domain, filepath, cap, DomainBlockResizeBytes)
}
