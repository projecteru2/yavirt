package domain

import (
	"encoding/xml"
	"path/filepath"
	"time"

	_ "embed"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/template"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	// InterfaceEthernet .
	InterfaceEthernet = "ethernet"
	// InterfaceBridge .
	InterfaceBridge = "bridge"
)

var (
	//go:embed templates/guest.xml
	guestXML string
	//go:embed templates/disk.xml
	diskXML string
)

// Domain .
type Domain interface { //nolint
	CheckShutoff() error
	GetUUID() (string, error)
	GetConsoleTtyname() (string, error)
	AttachVolume(filepath, devName string) (st libvirt.DomainState, err error)
	AmplifyVolume(filepath string, cap uint64) error
	Define() error
	Undefine() error
	Shutdown(force bool) error
	Boot() error
	Suspend() error
	Resume() error
	SetSpec(cpu int, mem int64) error
	GetState() (libvirt.DomainState, error)
}

// VirtDomain .
type VirtDomain struct {
	guest *models.Guest
	virt  libvirt.Libvirt
}

// New .
func New(guest *models.Guest, virt libvirt.Libvirt) *VirtDomain {
	return &VirtDomain{
		guest: guest,
		virt:  virt,
	}
}

// XML .
type XML struct {
	Name    string `xml:"name"`
	Devices struct {
		Channel []struct {
			Source struct {
				Path string `xml:"path,attr"`
			} `xml:"source"`
			Alias struct {
				Name string `xml:"name,attr"`
			} `xml:"alias"`
		} `xml:"channel"`
	} `xml:"devices"`
}

// Define .
func (d *VirtDomain) Define() error {
	buf, err := d.render()
	if err != nil {
		return errors.Trace(err)
	}

	dom, err := d.virt.DefineDomain(string(buf))
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Trace(err)
	case st == libvirt.DomainShutoff:
		return nil
	default:
		return types.NewDomainStatesErr(st, libvirt.DomainShutoff)
	}
}

// Boot .
func (d *VirtDomain) Boot() error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	var expState = libvirt.DomainShutoff
	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 5

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Trace(err)

		case st == libvirt.DomainRunning:
			return nil

		case st == expState:
			// Actually, dom.Create() means launch a defined domain.
			if err := dom.Create(); err != nil {
				return errors.Trace(err)
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

// Shutdown .
func (d *VirtDomain) Shutdown(force bool) error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	var expState = libvirt.DomainRunning

	shut := d.graceShutdown
	if force {
		shut = d.forceShutdown
	}

	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 5

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Trace(err)

		case st == libvirt.DomainShutoff:
			return nil

		case st == libvirt.DomainShutting:
			// It's shutting now, waiting to be shutoff.
			continue

		case st == libvirt.DomainPaused:
			fallthrough
		case st == expState:
			if err := shut(dom); err != nil {
				return errors.Trace(err)
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

func (d *VirtDomain) graceShutdown(dom libvirt.Domain) error {
	return dom.ShutdownFlags(libvirt.DomainShutdownDefault)
}

func (d *VirtDomain) forceShutdown(dom libvirt.Domain) error {
	return dom.DestroyFlags(libvirt.DomainDestroyDefault)
}

// CheckShutoff .
func (d *VirtDomain) CheckShutoff() error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Trace(err)
	case st != libvirt.DomainShutoff:
		return types.NewDomainStatesErr(st, libvirt.DomainShutoff)
	default:
		return nil
	}
}

// Suspend .
func (d *VirtDomain) Suspend() error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	var expState = libvirt.DomainRunning
	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 3

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Trace(err)

		case st == libvirt.DomainPaused:
			return nil

		case st == expState:
			if err := dom.Suspend(); err != nil {
				return errors.Trace(err)
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

// Resume .
func (d *VirtDomain) Resume() error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	var expState = libvirt.DomainPaused
	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 3

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Trace(err)

		case st == libvirt.DomainRunning:
			return nil

		case st == expState:
			if err := dom.Resume(); err != nil {
				return errors.Trace(err)
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

// Undefine .
func (d *VirtDomain) Undefine() error {
	dom, err := d.lookup()
	if err != nil {
		if errors.IsDomainNotExistsErr(err) {
			return nil
		}
		return errors.Trace(err)
	}
	defer dom.Free()

	var expState = libvirt.DomainShutoff
	switch st, err := dom.GetState(); {
	case err != nil:
		if errors.IsDomainNotExistsErr(err) {
			return nil
		}
		return errors.Trace(err)

	case st == libvirt.DomainPaused:
		fallthrough
	case st == expState:
		return dom.UndefineFlags(libvirt.DomainUndefineManagedSave)

	default:
		return types.NewDomainStatesErr(st, expState)
	}
}

// GetUUID .
func (d *VirtDomain) GetUUID() (string, error) {
	dom, err := d.lookup()
	if err != nil {
		return "", errors.Trace(err)
	}
	defer dom.Free()
	return dom.GetUUIDString()
}

func (d *VirtDomain) render() ([]byte, error) {
	uuid, err := d.checkUUID(d.guest.DmiUUID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	sysVol, err := d.guest.SysVolume()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var args = map[string]any{
		"name":              d.guest.ID,
		"uuid":              uuid,
		"memory":            d.guest.MemoryInMiB(),
		"cpu":               d.guest.CPU,
		"gpus":              []string{},
		"sysvol":            sysVol.Filepath(),
		"gasock":            d.guest.SocketFilepath(),
		"datavols":          d.dataVols(d.guest.Vols),
		"interface":         d.getInterfaceType(),
		"pair":              d.guest.NetworkPairName(),
		"mac":               d.guest.MAC,
		"cache_passthrough": configs.Conf.VirtCPUCachePassthrough,
	}

	return template.Render(d.guestTemplateFilepath(), guestXML, args)
}

func (d *VirtDomain) checkUUID(raw string) (string, error) {
	if len(raw) < 1 {
		return utils.UUIDStr()
	}

	if err := utils.CheckUUID(raw); err != nil {
		return "", errors.Trace(err)
	}

	return raw, nil
}

func (d *VirtDomain) getInterfaceType() string {
	switch d.guest.NetworkMode {
	case vnet.NetworkCalico:
		return InterfaceEthernet
	default:
		return InterfaceBridge
	}
}

func (d *VirtDomain) dataVols(vols models.Volumes) []map[string]string {
	var dat = []map[string]string{}

	for i, v := range vols {
		if v.IsSys() {
			continue
		}

		dat = append(dat, map[string]string{
			"path": v.Filepath(),
			"dev":  v.GetDeviceName(i),
		})
	}

	return dat
}

// GetXMLString .
func (d *VirtDomain) GetXMLString() (xml string, err error) {
	dom, err := d.lookup()
	if err != nil {
		return
	}
	defer dom.Free()

	var flags libvirt.DomainXMLFlags
	return dom.GetXMLDesc(flags)
}

// GetConsoleTtyname .
func (d *VirtDomain) GetConsoleTtyname() (devname string, err error) {
	var dom libvirt.Domain
	if dom, err = d.lookup(); err != nil {
		return
	}
	defer dom.Free()

	expState := libvirt.DomainRunning
	switch st, err := dom.GetState(); {
	case err != nil:
		return "", errors.Trace(err)

	case st != expState:
		return "", types.NewDomainStatesErr(st, expState)
	}

	x, err := d.GetXMLString()
	if err != nil {
		return
	}
	domainXML := &XML{}
	if err = xml.Unmarshal([]byte(x), domainXML); err != nil {
		return
	}
	for _, c := range domainXML.Devices.Channel {
		if c.Alias.Name == "channel0" {
			return c.Source.Path, nil
		}
	}
	return "", errors.Errorf("channel0 not found")
}

// SetSpec .
func (d *VirtDomain) SetSpec(cpu int, mem int64) error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()

	if err := d.setCPU(cpu, dom); err != nil {
		return errors.Trace(err)
	}

	return d.setMemory(mem, dom)
}

func (d *VirtDomain) setCPU(cpu int, dom libvirt.Domain) error {
	switch {
	case cpu < 0:
		return errors.Annotatef(errors.ErrInvalidValue, "invalid CPU num: %d", cpu)
	case cpu == 0:
		return nil
	}

	flag := libvirt.DomainVcpuConfig
	// Doesn't set with both Maximum and Current simultaneously.
	if err := dom.SetVcpusFlags(uint(cpu), flag|libvirt.DomainVcpuMaximum); err != nil {
		return errors.Trace(err)
	}
	return dom.SetVcpusFlags(uint(cpu), flag|libvirt.DomainVcpuCurrent)
}

func (d *VirtDomain) setMemory(mem int64, dom libvirt.Domain) error {
	if mem < configs.Conf.MinMemory || mem > configs.Conf.MaxMemory {
		return errors.Annotatef(errors.ErrInvalidValue,
			"invalid memory: %d, it shoule be [%d, %d]",
			mem, configs.Conf.MinMemory, configs.Conf.MaxMemory)
	}

	// converts bytes unit to kilobytes
	mem >>= 10

	flag := libvirt.DomainMemConfig
	if err := dom.SetMemoryFlags(uint64(mem), flag|libvirt.DomainMemMaximum); err != nil {
		return errors.Trace(err)
	}
	return dom.SetMemoryFlags(uint64(mem), flag|libvirt.DomainMemCurrent)
}

// AttachVolume .
func (d *VirtDomain) AttachVolume(filepath, devName string) (st libvirt.DomainState, err error) {
	var dom libvirt.Domain
	if dom, err = d.lookup(); err != nil {
		return
	}
	defer dom.Free()

	var buf []byte
	if buf, err = d.renderAttachVolumeXML(filepath, devName); err != nil {
		return
	}

	return dom.AttachVolume(string(buf))
}

func (d *VirtDomain) renderAttachVolumeXML(filepath, devName string) ([]byte, error) {
	args := map[string]any{
		"path": filepath,
		"dev":  devName,
	}
	return template.Render(d.diskTemplateFilepath(), diskXML, args)
}

// GetState .
func (d *VirtDomain) GetState() (libvirt.DomainState, error) {
	dom, err := d.lookup()
	if err != nil {
		return libvirt.DomainNoState, errors.Trace(err)
	}
	defer dom.Free()
	return dom.GetState()
}

// AmplifyVolume .
func (d *VirtDomain) AmplifyVolume(filepath string, cap uint64) error {
	dom, err := d.lookup()
	if err != nil {
		return errors.Trace(err)
	}
	defer dom.Free()
	return dom.AmplifyVolume(filepath, cap)
}

func (d *VirtDomain) lookup() (libvirt.Domain, error) {
	return d.virt.LookupDomain(d.guest.ID)
}

func (d *VirtDomain) diskTemplateFilepath() string {
	return filepath.Join(configs.Conf.VirtTmplDir, "disk.xml")
}

func (d *VirtDomain) guestTemplateFilepath() string {
	return filepath.Join(configs.Conf.VirtTmplDir, "guest.xml")
}

// GetState .
func GetState(name string, virt libvirt.Libvirt) (libvirt.DomainState, error) {
	dom, err := virt.LookupDomain(name)
	if err != nil {
		return libvirt.DomainNoState, errors.Trace(err)
	}
	defer dom.Free()
	return dom.GetState()
}
