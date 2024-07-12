package domain

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/antchfx/xmlquery"
	"github.com/cockroachdb/errors"
	pciaddr "github.com/jaypipes/ghw/pkg/pci/address"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/eru/resources"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/template"
	"github.com/projecteru2/yavirt/internal/vmcache"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/samber/lo"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
	"libvirt.org/go/libvirtxml"
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
	//go:embed templates/hostdev.xml
	hostdevXML string
)

// Domain .
type Domain interface { //nolint
	Lookup() (libvirt.Domain, error)
	CheckShutoff() error
	CheckRunning() error
	GetUUID() (string, error)
	GetConsoleTtyname() (string, error)
	OpenConsole(devname string, flages types.OpenConsoleFlags) (*libvirt.Console, error)
	ReplaceSysVolume(diskXML string) error
	AttachVolume(buf []byte) (st libvirt.DomainState, err error)
	DetachVolume(dev string) (st libvirt.DomainState, err error)
	AttachGPU(prod string, count int) (st libvirt.DomainState, err error)
	DetachGPU(prod string, count int) (st libvirt.DomainState, err error)
	AmplifyVolume(filepath string, cap uint64) error
	Define() error
	Undefine() error
	Shutdown(ctx context.Context, force bool) error
	Boot(ctx context.Context) error
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

// Define .
func (d *VirtDomain) Define() error {
	ctx := context.TODO()
	logger := log.WithFunc("VirtDomain.Define").WithField("guest", d.guest.ID)

	logger.Debugf(ctx, "GPU engine params: %v", d.guest.GPUEngineParams)
	// if gpu resource is needed, we need to lock gpu resources here
	// and unlock after the domain is defined
	if d.guest.GPUEngineParams.Count() > 0 {
		resources.GetManager().LockGPU()
		defer resources.GetManager().UnlockGPU()

		// Updating the domain cache is necessary in this context. Consider the following scenario:
		// We do not update the vmcache here because the event-driven update of vmcache may experience delays.
		// Consequently, after unlocking the GPU locker, the vmcache may not have been updated.
		// In such cases, the next GPU allocation may inadvertently select GPUs that are already in use by this VM.
		// While this scenario is rare, it can occur.
		defer func() {
			logger.Debugf(ctx, " -------------- %s GPU addresses: %v", d.guest.ID, vmcache.FetchGPUAddrs())
			if err := vmcache.UpdateDomain(d.guest.ID); err != nil {
				log.Errorf(ctx, err, "[Define] failed to update domain cache")
			}
			logger.Debugf(ctx, " +++++++++++++ %s GPU addresses: %v", d.guest.ID, vmcache.FetchGPUAddrs())
		}()
	}

	buf, err := d.render()
	if err != nil {
		return errors.Wrap(err, "")
	}

	dom, err := d.virt.DefineDomain(string(buf))
	if err != nil {
		return errors.Wrap(err, "")
	}

	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Wrap(err, "")
	case st == libvirt.DomainShutoff:
		return nil
	default:
		return types.NewDomainStatesErr(st, libvirt.DomainShutoff)
	}
}

// Boot .
func (d *VirtDomain) Boot(ctx context.Context) error {
	logger := log.WithFunc("VirtDomain.Boot")
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer func() {
		_ = dom.SetAutostart(true)
		if err := dom.SetMemoryStatsPeriod(configs.Conf.MemStatsPeriod, false, false); err != nil {
			logger.Warnf(ctx, "failed to set memory stats period: %v", err)
		}
	}()

	domName, _ := dom.GetName()
	var expState = libvirt.DomainShutoff
	for i := 0; ; i++ {
		timeout := time.Duration(i%5) * time.Second

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeout):
			switch st, err := dom.GetState(); {
			case err != nil:
				return errors.Wrap(err, "")

			case st == libvirt.DomainRunning:
				return nil

			case st == expState:
				// Actually, dom.Create() means launch a defined domain.
				if err := dom.Create(); err != nil {
					logger.Debugf(ctx, "create domain failed,dom name : %s , err: %s", domName, err.Error())
					return errors.Wrap(err, "")
				}
				logger.Infof(ctx, "create domain success, dom name : %s", domName)
				continue

			default:
				return types.NewDomainStatesErr(st, expState)
			}
		}
	}
}

// Shutdown .
func (d *VirtDomain) Shutdown(ctx context.Context, force bool) error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	var expState = libvirt.DomainRunning

	shut := d.graceShutdown
	if force {
		shut = d.forceShutdown
	}

	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(time.Second * time.Duration(i))
			i %= 5

			switch st, err := dom.GetState(); {
			case err != nil:
				return errors.Wrap(err, "")

			case st == libvirt.DomainShutoff:
				return nil

			case st == libvirt.DomainShutting:
				// It's shutting now, waiting to be shutoff.
				continue

			case st == libvirt.DomainPaused:
				fallthrough
			case st == expState:
				if err := shut(dom); err != nil {
					return errors.Wrap(err, "")
				}
				continue

			default:
				return types.NewDomainStatesErr(st, expState)
			}
		}
	}
}

func (d *VirtDomain) graceShutdown(dom libvirt.Domain) error {
	return dom.ShutdownFlags(libvirt.DomainShutdownFlags(libvirt.DomainShutdownDefault))
}

func (d *VirtDomain) forceShutdown(dom libvirt.Domain) error {
	return dom.DestroyFlags(libvirt.DomainDestroyDefault)
}

// CheckShutoff .
func (d *VirtDomain) CheckShutoff() error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Wrap(err, "")
	case st != libvirt.DomainShutoff:
		return types.NewDomainStatesErr(st, libvirt.DomainShutoff)
	default:
		return nil
	}
}

// CheckRunning .
func (d *VirtDomain) CheckRunning() error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	switch st, err := dom.GetState(); {
	case err != nil:
		return errors.Wrap(err, "")
	case st != libvirt.DomainRunning:
		return types.NewDomainStatesErr(st, libvirt.DomainRunning)
	default:
		return nil
	}
}

// Suspend .
func (d *VirtDomain) Suspend() error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	var expState = libvirt.DomainRunning
	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 3

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Wrap(err, "")

		case st == libvirt.DomainPaused:
			return nil

		case st == expState:
			if err := dom.Suspend(); err != nil {
				return errors.Wrap(err, "")
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

// Resume .
func (d *VirtDomain) Resume() error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	var expState = libvirt.DomainPaused
	for i := 0; ; i++ {
		time.Sleep(time.Second * time.Duration(i))
		i %= 3

		switch st, err := dom.GetState(); {
		case err != nil:
			return errors.Wrap(err, "")

		case st == libvirt.DomainRunning:
			return nil

		case st == expState:
			if err := dom.Resume(); err != nil {
				return errors.Wrap(err, "")
			}
			continue

		default:
			return types.NewDomainStatesErr(st, expState)
		}
	}
}

// Undefine .
func (d *VirtDomain) Undefine() error {
	dom, err := d.Lookup()
	if err != nil {
		if terrors.IsDomainNotExistsErr(err) {
			return nil
		}
		return errors.Wrap(err, "")
	}

	var expState = libvirt.DomainShutoff
	switch st, err := dom.GetState(); {
	case err != nil:
		if terrors.IsDomainNotExistsErr(err) {
			return nil
		}
		return errors.Wrap(err, "")

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
	dom, err := d.Lookup()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return dom.GetUUIDString()
}

func (d *VirtDomain) render() ([]byte, error) {
	uuid, err := d.checkUUID(d.guest.DmiUUID)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	sysVol, err := d.guest.SysVolume()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	sysVolXML, err := sysVol.GenerateXML()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	dataVols, err := d.dataVols()
	if err != nil {
		return nil, err
	}
	metadataXML, err := d.metadataXML()
	if err != nil {
		return nil, err
	}
	ciXML, cdromSrcXML, err := d.cloudInitXML()
	if err != nil {
		return nil, err
	}
	vncXML, err := d.vncConfig()
	if err != nil {
		return nil, err
	}

	var gpus []map[string]string
	if d.guest.GPUEngineParams.Count() > 0 {
		gpus, err = d.gpus()
		if err != nil {
			return nil, err
		}
	}
	hostDirs, err := d.hostDirs()
	if err != nil {
		return nil, err
	}
	var args = map[string]any{
		"name":              d.guest.ID,
		"uuid":              uuid,
		"memory":            d.guest.MemoryInMiB(),
		"cpu":               d.guest.CPU,
		"gpus":              gpus,
		"host_dirs":         hostDirs,
		"sysvol":            string(sysVolXML),
		"datavols":          dataVols,
		"interface":         d.getInterfaceType(),
		"pair":              d.guest.NetworkPairName(),
		"mac":               d.guest.MAC,
		"bandwidth":         d.networkBandwidth(),
		"cache_passthrough": configs.Conf.VirtCPUCachePassthrough,
		"metadata_xml":      metadataXML,
		"cloud_init_xml":    ciXML,
		"cdrom_src_xml":     cdromSrcXML,
		"vnc":               vncXML,
	}

	return template.Render(d.guestTemplateFilepath(), guestXML, args)
}

type AppMetadata struct {
	ID       int64  `json:"id"`
	SID      string `json:"sid"`
	Name     string `json:"name"`
	From     string `json:"from"`
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
}

func (d *VirtDomain) metadataXML() (string, error) {
	bs, ok := d.guest.JSONLabels["instance/metadata"]
	if !ok {
		return "", nil
	}
	obj := AppMetadata{}
	if err := json.Unmarshal([]byte(bs), &obj); err != nil {
		return "", errors.Wrap(err, "")
	}
	meta := types.CustomDomainMetadata{
		App: types.App{
			NS:   "https://eru.org/v1",
			From: obj.From,
			Owner: types.AppOwner{
				UserID:   fmt.Sprintf("%d", obj.UserID),
				UserName: obj.UserName,
			},
			Name: types.AppName{
				Name: obj.Name,
			},
			ID: types.AppID{
				SID: obj.SID,
				ID:  fmt.Sprintf("%d", obj.ID),
			},
		},
	}
	if len(d.guest.IPNets) > 0 {
		meta.App.IP.IP = d.guest.IPNets[0].IPv4()
	}
	xmlBS, err := xml.Marshal(meta)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return string(xmlBS), nil
}

func (d *VirtDomain) cloudInitXML() (string, string, error) {
	// for network
	obj, err := d.guest.GenCloudInit()
	if err != nil {
		return "", "", errors.Wrap(err, "")
	}
	log.Debugf(context.TODO(), "cloud-init: %v", obj)
	var (
		ciXML       string
		cdromSrcXML string
	)
	switch {
	case obj.URL != "":
		ciXML = fmt.Sprintf("<entry name='serial'>ds=nocloud-net;s=%s</entry>", obj.URL)
	case obj.Username != "" || obj.Password != "":
		output := filepath.Join(configs.Conf.VirtCloudInitDir, fmt.Sprintf("%s.iso", d.guest.ID))
		if err := obj.GenerateISO(output); err != nil {
			return "", "", err
		}
		ciXML = "<entry name='serial'>ds=nocloud</entry>"
		cdromSrcXML = fmt.Sprintf("<source file='%s' />", output)
	default:
		return "", "", errors.New("invalid cloud-init config")
	}
	return ciXML, cdromSrcXML, nil
}

func (d *VirtDomain) checkUUID(raw string) (string, error) {
	if len(raw) < 1 {
		return utils.UUIDStr()
	}

	if err := utils.CheckUUID(raw); err != nil {
		return "", errors.Wrap(err, "")
	}

	return raw, nil
}

func (d *VirtDomain) getInterfaceType() string {
	switch d.guest.NetworkMode {
	case network.CalicoMode:
		return InterfaceEthernet
	default:
		return InterfaceBridge
	}
}

func (d *VirtDomain) dataVols() ([]string, error) {
	vols := d.guest.Vols
	var dat = []string{}

	for _, v := range vols {
		if v.IsSys() {
			continue
		}
		buf, err := v.GenerateXML()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		dat = append(dat, string(buf))
	}
	return dat, nil
}

func allocGPUs(eParams *gputypes.EngineParams) ([]map[string]string, error) {
	infos, err := resources.GetManager().AllocGPU(eParams)
	if err != nil {
		return nil, err
	}
	res := lo.Map(infos, func(info types.GPUInfo, _ int) map[string]string {
		addr := pciaddr.FromString(info.Address)
		r := map[string]string{
			"domain":   addr.Domain,
			"bus":      addr.Bus,
			"slot":     addr.Device,
			"function": addr.Function,
		}
		return r
	})
	return res, nil
}
func (d *VirtDomain) gpus() ([]map[string]string, error) {
	return allocGPUs(d.guest.GPUEngineParams)
}

func (d *VirtDomain) hostDirs() ([]map[string]string, error) {
	ss, ok := d.guest.JSONLabels["instance/host-dirs"]
	if !ok {
		return nil, nil
	}
	parts := strings.FieldsFunc(ss, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	ans := make([]map[string]string, 0, len(parts))
	for _, p := range parts {
		switch parts2 := strings.Split(p, ":"); len(parts2) {
		case 2:
			ans = append(ans, map[string]string{
				"src": parts2[0],
				"dst": parts2[1],
			})
		default:
			return nil, fmt.Errorf("invalid host dir: %s", p)
		}
	}
	return ans, nil
}

type vncConfig struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

func (d *VirtDomain) vncConfig() (string, error) {
	bs, ok := d.guest.JSONLabels["instance/vnc"]
	if !ok {
		return "", nil
	}
	obj := vncConfig{}
	if err := json.Unmarshal([]byte(bs), &obj); err != nil {
		return "", errors.Wrap(err, "")
	}
	portCfg := "port='-1' autoport='yes'"
	if obj.Port > 0 {
		portCfg = fmt.Sprintf("port='%d'", obj.Port)
	}
	passwdCfg := ""
	if obj.Password != "" {
		passwdCfg = fmt.Sprintf("passwd='%s'", obj.Password)
	}
	vncXML := fmt.Sprintf(`<graphics type='vnc' %s listen='0.0.0.0' %s/>`, portCfg, passwdCfg)
	return vncXML, nil
}

func (d *VirtDomain) networkBandwidth() map[string]string {
	// the Unit of libvirt is kbyte/s
	// the default settings is avg: 2Gbps, peak: 3Gbps
	// 1Gbps=1000Mbps=1000000Kbps=1000000000bit
	ans := map[string]string{
		"average": fmt.Sprintf("%d", 2000000/8),
		"peak":    fmt.Sprintf("%d", 3000000/8),
	}
	ss, ok := d.guest.JSONLabels["instance/nic-bandwidth"]
	if !ok {
		return ans
	}

	bandwidth := map[string]int64{}
	err := json.Unmarshal([]byte(ss), &bandwidth)
	if err != nil {
		// just print log and use default values.
		log.Warnf(context.TODO(), "Invalid bandwidth label: %s", ss)
	} else {
		if v, ok := bandwidth["average"]; ok {
			ans["average"] = fmt.Sprintf("%d", v/8000)
		}
		if v, ok := bandwidth["peak"]; ok {
			ans["peak"] = fmt.Sprintf("%d", v/8000)
		}
	}
	return ans
}

// GetXMLString .
func (d *VirtDomain) GetXMLString() (xml string, err error) {
	dom, err := d.Lookup()
	if err != nil {
		return
	}

	var flags libvirt.DomainXMLFlags
	return dom.GetXMLDesc(flags)
}

// GetConsoleTtyname .
func (d *VirtDomain) GetConsoleTtyname() (devname string, err error) {
	x, err := d.GetXMLString()
	if err != nil {
		return
	}
	doc, err := xmlquery.Parse(strings.NewReader(x))
	if err != nil {
		return
	}
	aliasNode := xmlquery.FindOne(doc, "//devices/console[2]/alias")
	if aliasNode != nil {
		return aliasNode.SelectAttr("name"), nil
	}
	return "", nil
}

func (d *VirtDomain) OpenConsole(devname string, flags types.OpenConsoleFlags) (*libvirt.Console, error) {
	dom, err := d.Lookup()
	if err != nil {
		return nil, err
	}
	return dom.OpenConsole(devname, &flags.ConsoleFlags)
}

// SetSpec .
func (d *VirtDomain) SetSpec(cpu int, mem int64) error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := d.setCPU(cpu, dom); err != nil {
		return errors.Wrap(err, "")
	}

	return d.setMemory(mem, dom)
}

func (d *VirtDomain) setCPU(cpu int, dom libvirt.Domain) error {
	switch {
	case cpu < 0:
		return errors.Wrapf(terrors.ErrInvalidValue, "invalid CPU num: %d", cpu)
	case cpu == 0:
		return nil
	}

	flag := libvirt.DomainVcpuConfig
	// Doesn't set with both Maximum and Current simultaneously.
	if err := dom.SetVcpusFlags(uint(cpu), flag|libvirt.DomainVcpuMaximum); err != nil {
		return errors.Wrap(err, "")
	}
	return dom.SetVcpusFlags(uint(cpu), flag|libvirt.DomainVcpuCurrent)
}

func (d *VirtDomain) setMemory(mem int64, dom libvirt.Domain) error {
	if mem < configs.Conf.Resource.MinMemory || mem > configs.Conf.Resource.MaxMemory {
		return errors.Wrapf(terrors.ErrInvalidValue,
			"invalid memory: %d, it shoule be [%d, %d]",
			mem, configs.Conf.Resource.MinMemory, configs.Conf.Resource.MaxMemory)
	}

	// converts bytes unit to kilobytes
	mem >>= 10

	flag := libvirt.DomainMemConfig
	if err := dom.SetMemoryFlags(uint64(mem), flag|libvirt.DomainMemMaximum); err != nil {
		return errors.Wrap(err, "")
	}
	return dom.SetMemoryFlags(uint64(mem), flag|libvirt.DomainMemCurrent)
}

func (d *VirtDomain) ReplaceSysVolume(diskXML string) error {
	xmldoc, err := d.GetXMLString()
	if err != nil {
		return errors.Wrapf(err, "failed to get domain xml of guest %s", d.guest.ID)
	}
	domcfg := &libvirtxml.Domain{}
	if err = domcfg.Unmarshal(xmldoc); err != nil {
		return errors.Wrapf(err, "failed to unmarshal domain xml of guest %s", d.guest.ID)
	}
	sysDisk := &libvirtxml.DomainDisk{}
	if err := sysDisk.Unmarshal(diskXML); err != nil {
		return errors.Wrapf(err, "faied to unmarshal disk xml")
	}
	domcfg.Devices.Disks[0] = *sysDisk
	newXMLDoc, err := domcfg.Marshal()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal new domain xml for guest %s", d.guest.ID)
	}
	if _, err := d.virt.DefineDomain(newXMLDoc); err != nil {
		return errors.Wrapf(err, "failed define domain for guest %s", d.guest.ID)
	}

	return nil
}

// AttachVolume .
func (d *VirtDomain) AttachVolume(buf []byte) (st libvirt.DomainState, err error) {
	var dom libvirt.Domain
	if dom, err = d.Lookup(); err != nil {
		return
	}
	return dom.AttachDevice(string(buf))
}

func (d *VirtDomain) DetachVolume(devPath string) (st libvirt.DomainState, err error) {
	x, err := d.GetXMLString()
	if err != nil {
		return
	}
	dev := filepath.Base(devPath)
	doc, err := xmlquery.Parse(strings.NewReader(x))
	if err != nil {
		return
	}
	node := xmlquery.FindOne(doc, fmt.Sprintf("//devices/disk[target[@dev='%s']]", dev))
	if node == nil {
		err = errors.New("can't find device")
		return
	}

	xml := node.OutputXML(true)
	log.Infof(context.TODO(), "Detach volume, device(%s) xml: %s", devPath, xml)
	var dom libvirt.Domain
	if dom, err = d.Lookup(); err != nil {
		return
	}
	return dom.DetachDevice(xml)
}

// AttachGPU attaches new GPUs to guest.
func (d *VirtDomain) AttachGPU(prod string, count int) (st libvirt.DomainState, err error) {
	logger := log.WithFunc("AttachGPU")
	var dom libvirt.Domain
	if dom, err = d.Lookup(); err != nil {
		return
	}

	resources.GetManager().LockGPU()
	defer resources.GetManager().UnlockGPU()

	// Updating the domain cache is necessary in this context. Consider the following scenario:
	// We do not update the vmcache here because the event-driven update of vmcache may experience delays.
	// Consequently, after unlocking the GPU locker, the vmcache may not have been updated.
	// In such cases, the next GPU allocation may inadvertently select GPUs that are already in use by this VM.
	// While this scenario is rare, it can occur.
	defer func() {
		if err := vmcache.UpdateDomain(d.guest.ID); err != nil {
			logger.Errorf(context.TODO(), err, "failed to update domain cache")
		}
	}()
	eParams := &gputypes.EngineParams{
		ProdCountMap: map[string]int{
			prod: count,
		},
	}
	infos, err := allocGPUs(eParams)
	if err != nil {
		return
	}
	var buf []byte
	for _, info := range infos {
		buf, err = template.Render(d.hostdevTemplateFilepath(), hostdevXML, info)
		if err != nil {
			return 0, err
		}

		if st, err = dom.AttachDevice(string(buf)); err != nil {
			return st, err
		}
	}
	return
}

func extractHostdevXML(doc *xmlquery.Node, gaddr string) (string, error) {
	ctx := context.TODO()
	logger := log.WithFunc("extractHostdevXML")

	addr := pciaddr.FromString(gaddr)
	xpathFmt := "//devices/hostdev[source[address[@domain='0x%s' and @bus='0x%s' and @slot='0x%s' and @function='0x%s']]]"
	node := xmlquery.FindOne(doc, fmt.Sprintf(xpathFmt, addr.Domain, addr.Bus, addr.Device, addr.Function))
	if node == nil {
		return "", errors.Errorf("can't find device, pciaddr: %s", gaddr)
	}
	logger.Debugf(ctx, "Detach gpu, device(%s) xml: %s", gaddr, node.OutputXML(true))

	xml := node.OutputXML(true)
	logger.Infof(ctx, "Detach gpu, device(%s) xml: %s", gaddr, xml)
	return xml, nil
}

// <hostdev mode='subsystem' type='pci' managed='yes'>
//
//	<source>
//	<address domain='0x{{.domain}}' bus='0x{{.bus}}' slot='0x{{.slot}}' function='0x{{.function}}'/>
//	</source>
//
// </hostdev>
func (d *VirtDomain) DetachGPU(_ string, count int) (st libvirt.DomainState, err error) {
	defer func() {
		if err := vmcache.UpdateDomain(d.guest.ID); err != nil {
			log.Errorf(context.TODO(), err, "[DetachGPU] failed to update domain cache")
		}
	}()
	var dom libvirt.Domain
	if dom, err = d.Lookup(); err != nil {
		return
	}

	var flags libvirt.DomainXMLFlags
	x, err := dom.GetXMLDesc(flags)

	if err != nil {
		return
	}
	doc, err := xmlquery.Parse(strings.NewReader(x))
	if err != nil {
		return
	}
	entry := vmcache.FetchDomainEntry(d.guest.ID)
	if count > len(entry.GPUAddrs) {
		count = len(entry.GPUAddrs)
	}
	for i := 0; i < count; i++ {
		gaddr := entry.GPUAddrs[i]
		// TODO check if the gaddr's product is equal to the product
		xml, err := extractHostdevXML(doc, gaddr)
		if err != nil {
			return 0, err
		}
		if st, err = dom.DetachDevice(xml); err != nil {
			return st, err
		}
	}
	return
}

// GetState .
func (d *VirtDomain) GetState() (libvirt.DomainState, error) {
	dom, err := d.Lookup()
	if err != nil {
		return libvirt.DomainNoState, errors.Wrap(err, "")
	}
	return dom.GetState()
}

// AmplifyVolume .
func (d *VirtDomain) AmplifyVolume(filepath string, cap uint64) error {
	dom, err := d.Lookup()
	if err != nil {
		return errors.Wrap(err, "")
	}
	return dom.AmplifyVolume(filepath, cap)
}

func (d *VirtDomain) Lookup() (libvirt.Domain, error) {
	return d.virt.LookupDomain(d.guest.ID)
}

func (d *VirtDomain) guestTemplateFilepath() string {
	return filepath.Join(configs.Conf.VirtTmplDir, "guest.xml")
}

func (d *VirtDomain) hostdevTemplateFilepath() string {
	return filepath.Join(configs.Conf.VirtTmplDir, "hostdev.xml")
}

// GetState .
func GetState(name string, virt libvirt.Libvirt) (libvirt.DomainState, error) {
	dom, err := virt.LookupDomain(name)
	if err != nil {
		return libvirt.DomainNoState, errors.Wrap(err, "")
	}
	return dom.GetState()
}
