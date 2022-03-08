package model

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	erucluster "github.com/projecteru2/core/cluster"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/handler"
)

// Guest indicates a virtual machine.
type Guest struct {
	*Generic

	ImageName        string            `json:"img"`
	ImageUser        string            `json:"img_user,omitempty"`
	HostName         string            `json:"host"`
	CPU              int               `json:"cpu"`
	Memory           int64             `json:"mem"`
	VolIDs           []string          `json:"vols"`
	IPNets           meta.IPNets       `json:"ips"`
	ExtraNetworks    Networks          `json:"extra_networks,omitempty"`
	NetworkMode      string            `json:"network,omitempty"`
	EnabledCalicoCNI bool              `json:"enabled_calico_cni,omitempty"`
	NetworkPair      string            `json:"network_pair,omitempty"`
	EndpointID       string            `json:"endpoint,omitempty"`
	MAC              string            `json:"mac"`
	JSONLabels       map[string]string `json:"labels"`

	LambdaOption *LambdaOptions `json:"lambda_option,omitempty"`
	LambdaStdin  bool           `json:"lambda_stdin,omitempty"`
	Host         *Host          `json:"-"`
	Img          Image          `json:"-"`
	Vols         Volumes        `json:"-"`
	IPs          IPs            `json:"-"`

	DmiUUID string `json:"-"`
}

// Check .
func (g *Guest) Check() error {
	if g.CPU < config.Conf.MinCPU || g.CPU > config.Conf.MaxCPU {
		return errors.Annotatef(errors.ErrInvalidValue,
			"invalid CPU num: %d, it should be [%d, %d]",
			g.CPU, config.Conf.MinCPU, config.Conf.MaxCPU)
	}

	if g.Memory < config.Conf.MinMemory || g.Memory > config.Conf.MaxMemory {
		return errors.Annotatef(errors.ErrInvalidValue,
			"invalie memory: %d, it shoule be [%d, %d]",
			g.Memory, config.Conf.MinMemory, config.Conf.MaxMemory)
	}

	if lab, exists := g.JSONLabels[erucluster.LabelMeta]; exists {
		obj := map[string]interface{}{}
		if err := util.JSONDecode([]byte(lab), &obj); err != nil {
			return errors.Annotatef(errors.ErrInvalidValue, "'%s' should be JSON format", lab)
		}
	}

	return g.Vols.Check()
}

// MetaKey .
func (g *Guest) MetaKey() string {
	return meta.GuestKey(g.ID)
}

// Create .
func (g *Guest) Create() error {
	g.Vols.genID()
	g.Vols.setGuestID(g.ID)
	g.VolIDs = g.Vols.ids()

	g.IPs.setGuestID(g.ID)

	var res = meta.Resources{g}
	res.Concate(g.Vols.resources())
	res.Concate(meta.Resources{newHostGuest(g.HostName, g.ID)})

	return meta.Create(res)
}

// AppendIPs .
func (g *Guest) AppendIPs(ips ...meta.IP) {
	g.IPs.append(ips...)
	g.IPNets = g.IPs.ipNets()
}

// ClearVols .
func (g *Guest) ClearVols() {
	g.Vols = g.Vols[:0]
	g.VolIDs = g.VolIDs[:0]
}

// RemoveVol .
func (g *Guest) RemoveVol(volID string) {
	n := g.Vols.Len()

	for i := n - 1; i >= 0; i-- {
		if g.Vols[i].ID != volID {
			continue
		}

		last := n - 1
		g.Vols[last], g.Vols[i] = g.Vols[i], g.Vols[last]
		g.VolIDs[last], g.VolIDs[i] = g.VolIDs[i], g.VolIDs[last]

		n--
	}

	g.Vols = g.Vols[:n]
	g.VolIDs = g.VolIDs[:n]
}

// AppendVols .
func (g *Guest) AppendVols(vols ...*Volume) error {
	if g.Vols.Len()+len(vols) > config.Conf.MaxVolumesCount {
		return errors.Annotatef(errors.ErrTooManyVolumes, "at most %d", config.Conf.MaxVolumesCount)
	}

	var res = Volumes(vols)
	res.setHostName(g.HostName)

	g.Vols.append(vols...)

	g.VolIDs = append(g.VolIDs, res.ids()...)

	return nil
}

func (g *Guest) SwitchVol(vol *Volume, idx int) error {
	if idx < 0 || idx >= g.Vols.Len() {
		return errors.Annotatef(errors.ErrInvalidValue, "must in range 0 to %d", g.Vols.Len()-1)
	}

	g.Vols[idx] = vol
	g.VolIDs[idx] = vol.ID

	return nil
}

// Load .
func (g *Guest) Load(host *Host, networkHandler handler.Handler) (err error) {
	g.Host = host

	if g.Img, err = LoadImage(g.ImageName, g.ImageUser); err != nil {
		return errors.Trace(err)
	}

	if g.Vols, err = LoadVolumes(g.VolIDs); err != nil {
		return errors.Trace(err)
	}

	if err = g.LoadIPs(networkHandler); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadIPs .
func (g *Guest) LoadIPs(networkHandler handler.Handler) (err error) {
	for _, ipn := range g.IPNets {
		ipn.Assigned = true
	}

	g.IPs, err = networkHandler.QueryIPs(g.IPNets)

	return
}

// Resize .
func (g *Guest) Resize(cpu int, mem int64) error {
	g.CPU = cpu
	g.Memory = mem
	return g.Save()
}

// Save updates metadata to persistence store.
func (g *Guest) Save() error {
	return meta.Save(meta.Resources{g})
}

// Delete .
func (g *Guest) Delete(force bool) error {
	if err := g.setStatus(StatusDestroyed, force); err != nil {
		return errors.Trace(err)
	}

	var keys = []string{
		g.MetaKey(),
		newHostGuest(g.HostName, g.ID).MetaKey(),
	}
	keys = append(keys, g.Vols.deleteKeys()...)

	var vers = map[string]int64{g.MetaKey(): g.GetVer()}
	for _, vol := range g.Vols {
		vers[vol.MetaKey()] = vol.GetVer()
	}

	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}

// SysVolume .
func (g *Guest) SysVolume() (*Volume, error) {
	for _, vol := range g.Vols {
		if vol.IsSys() {
			return vol, nil
		}
	}
	return nil, errors.ErrSysVolumeNotExists
}

// HealthCheck .
func (g *Guest) HealthCheck() (HealthCheck, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return HealthCheck{}, errors.Trace(err)
	}
	return hcb.healthCheck(g)
}

// PublishPorts .
func (g *Guest) PublishPorts() ([]int, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return []int{}, errors.Trace(err)
	}
	return hcb.publishPorts()
}

// CIDRs .
func (g *Guest) CIDRs() []string {
	cidrs := make([]string, len(g.IPNets))
	for i, ipn := range g.IPNets {
		cidrs[i] = ipn.CIDR()
	}
	return cidrs
}

// MemoryInMiB .
func (g *Guest) MemoryInMiB() int64 {
	return util.ConvToMB(g.Memory)
}

// SocketFilepath shows the socket filepath of the guest on the host.
func (g *Guest) SocketFilepath() string {
	var fn = fmt.Sprintf("%s.sock", g.ID)
	return filepath.Join(config.Conf.VirtSockDir, fn)
}

// NetworkPairName .
func (g *Guest) NetworkPairName() string {
	switch {
	case g.NetworkMode == vnet.NetworkCalico:
		fallthrough
	case len(g.NetworkPair) > 0:
		return g.NetworkPair

	default:
		return config.Conf.VirtBridge
	}
}

func newGuest() *Guest {
	return &Guest{
		Generic: newGeneric(),
		Vols:    Volumes{},
		IPs:     IPs{},
	}
}

// LoadGuest .
func LoadGuest(id string) (*Guest, error) {
	return manager.LoadGuest(id)
}

// CreateGuest .
func CreateGuest(opts types.GuestCreateOption, host *Host, vols []*Volume) (*Guest, error) {
	return manager.CreateGuest(opts, host, vols)
}

// NewGuest creates a new guest.
func NewGuest(host *Host, img Image) (*Guest, error) {
	return manager.NewGuest(host, img)
}

// GetNodeGuests gets all guests which belong to the node.
func GetNodeGuests(nodename string) ([]*Guest, error) {
	return manager.GetNodeGuests(nodename)
}

// GetAllGuests .
func GetAllGuests() ([]*Guest, error) {
	return manager.GetAllGuests()
}

// LoadGuest .
func (m *Manager) LoadGuest(id string) (*Guest, error) {
	g, err := m.NewGuest(nil, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	g.ID = id

	if err := meta.Load(g); err != nil {
		return nil, errors.Trace(err)
	}

	return g, nil
}

// CreateGuest .
func (m *Manager) CreateGuest(opts types.GuestCreateOption, host *Host, vols []*Volume) (*Guest, error) {
	img, err := LoadImage(opts.ImageName, opts.ImageUser)
	if err != nil {
		return nil, errors.Trace(err)
	}

	guest, err := m.NewGuest(host, img)
	if err != nil {
		return nil, errors.Trace(err)
	}

	guest.ID = idgen.Next()
	guest.CPU = opts.CPU
	guest.Memory = opts.Mem
	guest.DmiUUID = opts.DmiUUID
	guest.JSONLabels = opts.Labels

	if guest.NetworkMode == vnet.NetworkCalico {
		guest.EnabledCalicoCNI = config.Conf.EnabledCalicoCNI
	}

	if opts.Lambda {
		guest.LambdaOption = &LambdaOptions{
			Cmd:       opts.Cmd,
			CmdOutput: nil,
		}
	}

	if err := guest.AppendVols(vols...); err != nil {
		return nil, errors.Trace(err)
	}

	if err := guest.Check(); err != nil {
		return nil, errors.Trace(err)
	}

	if err := guest.Create(); err != nil {
		return nil, errors.Trace(err)
	}

	return guest, nil
}

// NewGuest creates a new guest.
func (m *Manager) NewGuest(host *Host, img Image) (*Guest, error) {
	var guest = newGuest()

	if host != nil {
		guest.Host = host
		guest.HostName = guest.Host.Name
		guest.NetworkMode = guest.Host.NetworkMode
	}

	if img != nil {
		guest.Img = img
		guest.ImageName = guest.Img.GetName()
		guest.ImageUser = guest.Img.GetUser()
		if err := guest.AppendVols(guest.Img.NewSysVolume()); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return guest, nil
}

// GetNodeGuests gets all guests which belong to the node.
func (m *Manager) GetNodeGuests(nodename string) ([]*Guest, error) {
	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	data, _, err := store.GetPrefix(ctx, meta.HostGuestsPrefix(nodename), 0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	guests := []*Guest{}
	for key := range data {
		parts := strings.Split(key, "/")
		gid := parts[len(parts)-1]
		if len(gid) < 1 {
			continue
		}

		g, err := m.LoadGuest(gid)
		if err != nil {
			return nil, errors.Trace(err)
		}

		guests = append(guests, g)
	}

	return guests, nil
}

// GetAllGuests .
func (m *Manager) GetAllGuests() ([]*Guest, error) {
	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	var data, vers, err = store.GetPrefix(ctx, meta.GuestsPrefix(), 0)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var guests = []*Guest{}

	for key, val := range data {
		var ver, exists = vers[key]
		if !exists {
			return nil, errors.Annotatef(errors.ErrKeyBadVersion, key)
		}

		var g = newGuest()
		if err := util.JSONDecode(val, g); err != nil {
			return nil, errors.Trace(err)
		}

		g.SetVer(ver)
		guests = append(guests, g)
	}

	return guests, nil
}
