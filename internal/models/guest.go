package models

import (
	"context"
	"encoding/json"
	"strings"

	erucluster "github.com/projecteru2/core/cluster"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/volume"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
	bdtypes "github.com/yuyang0/resource-bandwidth/bandwidth/types"
	gputypes "github.com/yuyang0/resource-gpu/gpu/types"
)

// Guest indicates a virtual machine.
type Guest struct {
	*meta.Generic

	ImageName       string                 `json:"img"`
	HostName        string                 `json:"host"`
	CPU             int                    `json:"cpu"`
	Memory          int64                  `json:"mem"`
	VolIDs          []string               `json:"vols"`
	GPUEngineParams *gputypes.EngineParams `json:"gpu_engine_params"`
	BDEngineParams  *bdtypes.EngineParams  `json:"bandwidth_engine_params"`
	IPNets          meta.IPNets            `json:"ips"`
	ExtraNetworks   Networks               `json:"extra_networks,omitempty"`
	NetworkMode     string                 `json:"network,omitempty"`
	NetworkPair     string                 `json:"network_pair,omitempty"`
	EndpointID      string                 `json:"endpoint,omitempty"`
	MAC             string                 `json:"mac"`
	MTU             int                    `json:"mtu"`
	JSONLabels      map[string]string      `json:"labels"`

	LambdaOption *LambdaOptions  `json:"lambda_option,omitempty"`
	LambdaStdin  bool            `json:"lambda_stdin,omitempty"`
	Host         *Host           `json:"-"`
	Img          *vmitypes.Image `json:"-"`
	Vols         volFact.Volumes `json:"-"`
	IPs          IPs             `json:"-"`

	DmiUUID string `json:"-"`
}

// Check .
func (g *Guest) Check() error {
	if g.CPU < configs.Conf.Resource.MinCPU || g.CPU > configs.Conf.Resource.MaxCPU {
		return errors.Wrapf(terrors.ErrInvalidValue,
			"invalid CPU num: %d, it should be [%d, %d]",
			g.CPU, configs.Conf.Resource.MinCPU, configs.Conf.Resource.MaxCPU)
	}

	if g.Memory < configs.Conf.Resource.MinMemory || g.Memory > configs.Conf.Resource.MaxMemory {
		return errors.Wrapf(terrors.ErrInvalidValue,
			"invalie memory: %d, it shoule be [%d, %d]",
			g.Memory, configs.Conf.Resource.MinMemory, configs.Conf.Resource.MaxMemory)
	}

	if lab, exists := g.JSONLabels[erucluster.LabelMeta]; exists {
		obj := map[string]any{}
		if err := utils.JSONDecode([]byte(lab), &obj); err != nil {
			return errors.Wrapf(terrors.ErrInvalidValue, "'%s' should be JSON format", lab)
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
	g.Vols.GenID()
	g.Vols.SetDevice()
	g.Vols.SetGuestID(g.ID)
	g.VolIDs = g.Vols.IDs()

	g.IPs.setGuestID(g.ID)

	var res = meta.Resources{g}
	res.Concate(g.Vols.Resources())
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
		if g.Vols[i].GetID() != volID {
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
func (g *Guest) AppendVols(vols ...volume.Volume) error {
	if g.Vols.Len()+len(vols) > configs.Conf.Resource.MaxVolumesCount {
		return errors.Wrapf(terrors.ErrTooManyVolumes, "at most %d", configs.Conf.Resource.MaxVolumesCount)
	}

	var res = volFact.Volumes(vols)
	res.SetHostName(g.HostName)

	g.Vols = append(g.Vols, vols...)

	g.VolIDs = append(g.VolIDs, res.IDs()...)

	return nil
}

func (g *Guest) SwitchVol(vol volume.Volume, idx int) error {
	if idx < 0 || idx >= g.Vols.Len() {
		return errors.WithMessagef(terrors.ErrInvalidValue, "must in range 0 to %d", g.Vols.Len()-1)
	}

	g.Vols[idx] = vol
	g.VolIDs[idx] = vol.GetID()

	return nil
}

// Load .
func (g *Guest) Load(host *Host, networkHandler network.Driver, opts ...Option) (err error) {
	logger := log.WithFunc("Guest.Load")
	op := NewOp(opts...)
	g.Host = host

	if g.Vols, err = volFact.LoadVolumes(g.VolIDs); err != nil {
		return errors.WithMessagef(err, "failed to load volumes %v", g.VolIDs)
	}

	if err = g.LoadIPs(networkHandler); err != nil {
		return errors.WithMessage(err, "failed to load IPs")
	}

	if g.Img, err = vmiFact.LoadImage(context.TODO(), g.ImageName); err != nil {
		if op.IgnoreLoadImageErr {
			logger.Warnf(context.TODO(), "failed to load image %s: %s", g.ImageName, err)
		} else {
			return errors.Wrapf(terrors.ErrLoadImage, "failed to load image %s: %s", g.ImageName, err)
		}
	}
	return nil
}

// LoadIPs .
func (g *Guest) LoadIPs(networkHandler network.Driver) (err error) {
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
	if err := g.SetStatus(meta.StatusDestroyed, force); err != nil {
		return errors.WithMessagef(err, "Delete: failed to set status to %s", meta.StatusDestroyed)
	}

	var keys = []string{
		g.MetaKey(),
		newHostGuest(g.HostName, g.ID).MetaKey(),
	}
	keys = append(keys, g.Vols.MetaKeys()...)

	var vers = map[string]int64{g.MetaKey(): g.GetVer()}
	for _, vol := range g.Vols {
		vers[vol.MetaKey()] = vol.GetVer()
	}

	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}

// SysVolume .
func (g *Guest) SysVolume() (volume.Volume, error) {
	for _, vol := range g.Vols {
		if vol.IsSys() {
			return vol, nil
		}
	}
	return nil, terrors.ErrSysVolumeNotExists
}

// HealthCheck .
func (g *Guest) HealthCheck() (HealthCheck, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return HealthCheck{}, errors.Wrap(err, "")
	}
	return hcb.healthCheck(g)
}

// PublishPorts .
func (g *Guest) PublishPorts() ([]int, error) {
	hcb, err := g.healthCheckBridge()
	if err != nil {
		return []int{}, errors.Wrap(err, "")
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
	return utils.ConvToMB(g.Memory)
}

// NetworkPairName .
func (g *Guest) NetworkPairName() string {
	switch {
	case g.NetworkMode == network.CalicoMode:
		fallthrough
	case len(g.NetworkPair) > 0:
		return g.NetworkPair

	default:
		return configs.Conf.VirtBridge
	}
}

// Generate cloud-init config from guest.
func (g *Guest) GenCloudInit() (*types.CloudInitConfig, error) {
	cidr := g.IPNets[0].CIDR()
	gwAddr := g.IPNets[0].GatewayAddr()
	inSubnet := netx.InSubnet(gwAddr, cidr)
	obj := &types.CloudInitConfig{
		IFName: "ens5",
		CIDR:   cidr,
		MAC:    g.MAC,
		MTU:    g.MTU,
		DefaultGW: types.CloudInitGateway{
			IP:     gwAddr,
			OnLink: !inSubnet,
		},
	}
	if bs, ok := g.JSONLabels["instance/cloud-init"]; ok {
		if err := json.Unmarshal([]byte(bs), &obj); err != nil {
			return nil, errors.Wrap(err, "")
		}
	} else {
		obj.Username = configs.Conf.VMAuth.Username
		obj.Password = configs.Conf.VMAuth.Password
	}
	if obj.Hostname == "" {
		obj.Hostname = interutils.RandomString(10)
	}
	if obj.InstanceID == "" {
		obj.InstanceID = obj.Hostname
	}
	return obj, nil
}

func newGuest() *Guest {
	return &Guest{
		Generic: meta.NewGeneric(),
		Vols:    volFact.Volumes{},
		IPs:     IPs{},
	}
}

// LoadGuest .
func LoadGuest(id string) (*Guest, error) {
	g := newGuest()
	g.ID = id

	if err := meta.Load(g); err != nil {
		return nil, errors.WithMessagef(err, "load guest %s", id)
	}

	return g, nil
}

// CreateGuest .
func CreateGuest(opts types.GuestCreateOption, host *Host, vols []volume.Volume) (*Guest, error) {
	img, err := vmiFact.LoadImage(context.TODO(), opts.ImageName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load image %s", opts.ImageName)
	}

	var guest = newGuest()
	guest.Host = host
	guest.HostName = guest.Host.Name
	guest.NetworkMode = opts.Labels[network.ModeLabelKey]
	if guest.NetworkMode == "" {
		guest.NetworkMode = host.DefaultNetworkMode
		opts.Labels[network.ModeLabelKey] = guest.NetworkMode
	}
	guest.MTU = 1500

	guest.Img = img
	guest.ImageName = img.Fullname()
	// Create sys volume when user doesn't specify one
	if len(vols) == 0 || (!vols[0].IsSys()) {
		sysVol := local.NewSysVolume(img.VirtualSize, img.Fullname())
		if err := guest.AppendVols(sysVol); err != nil {
			return nil, errors.WithMessagef(err, "Create: failed to append volume %s", sysVol)
		}
	}

	guest.ID = idgen.Next()
	guest.CPU = opts.CPU
	guest.Memory = opts.Mem
	guest.DmiUUID = opts.DmiUUID
	guest.JSONLabels = opts.Labels

	if opts.Lambda {
		guest.LambdaOption = &LambdaOptions{
			Cmd:       opts.Cmd,
			CmdOutput: nil,
		}
	}
	log.Debugf(context.TODO(), "Resources: %v", opts.Resources)
	if bs, ok := opts.Resources["gpu"]; ok {
		var eParams gputypes.EngineParams
		if err := json.Unmarshal(bs, &eParams); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal gpu params")
		}
		guest.GPUEngineParams = &eParams
	}

	if bs, ok := opts.Resources["bandwidth"]; ok {
		var eParams bdtypes.EngineParams
		if err := json.Unmarshal(bs, &eParams); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal bandwidth params")
		}
		guest.BDEngineParams = &eParams
	}
	if err := guest.AppendVols(vols...); err != nil {
		return nil, errors.WithMessagef(err, "CreateGuest: failed to append volumes %v", vols)
	}

	if err := guest.Check(); err != nil {
		return nil, errors.WithMessagef(err, "CreateGuest: failed to check guest %v", guest)
	}

	if err := guest.Create(); err != nil {
		return nil, err
	}

	return guest, nil
}

// NewGuest creates a new guest.
func NewGuest(host *Host, img *vmitypes.Image) (*Guest, error) {
	var guest = newGuest()

	if host != nil {
		guest.Host = host
		guest.HostName = guest.Host.Name
		guest.NetworkMode = guest.Host.DefaultNetworkMode
	}

	if img != nil {
		guest.Img = img
		guest.ImageName = img.Fullname()

		sysVol := local.NewSysVolume(img.VirtualSize, img.Fullname())
		if err := guest.AppendVols(sysVol); err != nil {
			return nil, errors.WithMessagef(err, "NewGuest: failed to append volume %s", sysVol)
		}
	}

	return guest, nil
}

// GetNodeGuests gets all guests which belong to the node.
func GetNodeGuests(nodename string) ([]*Guest, error) {
	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	data, _, err := store.GetPrefix(ctx, meta.HostGuestsPrefix(nodename), 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prefix")
	}

	guests := []*Guest{}
	for key := range data {
		parts := strings.Split(key, "/")
		gid := parts[len(parts)-1]
		if len(gid) < 1 {
			continue
		}

		g, err := LoadGuest(gid)
		if err != nil {
			return nil, errors.WithMessagef(err, "GetNodeGuests: failed to load guest %s", gid)
		}

		guests = append(guests, g)
	}

	return guests, nil
}

// GetAllGuests .
func GetAllGuests() ([]*Guest, error) {
	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	var data, vers, err = store.GetPrefix(ctx, meta.GuestsPrefix(), 0)
	if err != nil {
		return nil, errors.WithMessagef(err, "GetAllGuests: failed to get prefix")
	}

	var guests = []*Guest{}

	for key, val := range data {
		var ver, exists = vers[key]
		if !exists {
			return nil, errors.Wrapf(terrors.ErrKeyBadVersion, key)
		}

		var g = newGuest()
		if err := utils.JSONDecode(val, g); err != nil {
			return nil, errors.Wrapf(err, "GetAllGuests: failed to decode guest %s", key)
		}

		g.SetVer(ver)
		guests = append(guests, g)
	}

	return guests, nil
}
