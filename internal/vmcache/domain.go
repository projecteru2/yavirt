package vmcache

import (
	"context"
	"encoding/xml"
	"fmt"
	"sync"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/dustin/go-humanize"
	"github.com/patrickmn/go-cache"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"libvirt.org/go/libvirtxml"
)

var (
	gVC *VMCache
)

const (
	libvirtSock    = "/var/run/libvirt/libvirt-sock"
	libvirtTimeout = 5 * time.Second
)

func newLibvirt() (*libvirt.Libvirt, error) {
	opts := []dialers.LocalOption{
		dialers.WithSocket(libvirtSock),
		dialers.WithLocalTimeout(libvirtTimeout),
	}
	dialer := dialers.NewLocal(opts...)
	l := libvirt.NewWithDialer(dialer)
	if err := l.Connect(); err != nil {
		return nil, err
	}
	return l, nil
}

type VMCache struct {
	mu                  sync.Mutex
	localDomainCache    map[string]*DomainCacheEntry
	statsCache          *cache.Cache
	statsUpdateInterval time.Duration
	watchers            *interutils.Watchers
}

func (vc *VMCache) NotifyEvent(e libvirt.DomainEventLifecycleMsg) {
	evt := intertypes.Event{
		ID:   e.Dom.Name,
		Type: intertypes.EventTypeGuest,
	}
	switch e.Event {
	case DomainEventStarted:
		evt.Op = intertypes.StartOp
	case DomainEventStopped:
		evt.Op = intertypes.DieOp
	default:
		return
	}
	vc.watchers.Watched(evt)
}

// in order to avoid data race, please don't change this type of value after it created
type DomainCacheEntry struct {
	Name     string
	UUID     string
	State    libvirt.DomainState
	VNCPort  int
	CPU      int
	Memory   int64
	GPUAddrs []string
	Schema   *libvirtxml.Domain

	AppID    string
	AppSID   string
	AppName  string
	IP       string
	EruName  string
	UserName string
	UserID   string
}

// we don't copy Schema, because it never changes
func (dce *DomainCacheEntry) Copy() *DomainCacheEntry {
	ans := *dce
	ans.GPUAddrs = make([]string, len(dce.GPUAddrs))
	copy(ans.GPUAddrs, dce.GPUAddrs)
	return &ans
}

func (dce *DomainCacheEntry) IsRunning() bool {
	return dce.State == libvirt.DomainRunning
}

func (dce *DomainCacheEntry) IsStopped() bool {
	return dce.State == libvirt.DomainShutoff
}

type DomainStatsResp struct {
	DomainCacheEntry
	Stats map[string]libvirt.TypedParam
}

func extractInfoFromXML(domcfg *libvirtxml.Domain) *DomainCacheEntry {
	logger := log.WithFunc("extractInfoFromXML")
	var meta intertypes.CustomDomainMetadata
	if domcfg.Metadata != nil {
		metadataStr := fmt.Sprintf("<metadata>%s</metadata>", domcfg.Metadata.XML)
		if err := xml.Unmarshal([]byte(metadataStr), &meta); err != nil {
			// ignore error of metadata
			logger.Warnf(context.TODO(), "failed to unmarshal metadata:%s", err)
		}
	}
	cpu := int(domcfg.VCPU.Value)
	memStr := fmt.Sprintf("%d %s", domcfg.Memory.Value, domcfg.Memory.Unit)
	memory, err := humanize.ParseBytes(memStr)
	if err != nil {
		logger.Errorf(context.TODO(), err, "failed to parse memory of %s", domcfg.Name)
	}
	// fetch vnc port
	var vncPort int
	for _, g := range domcfg.Devices.Graphics {
		if g.VNC != nil {
			vncPort = g.VNC.Port
			break
		}
	}
	// fetch GPU addrs
	var gpuAddrs []string //nolint
	for _, hd := range domcfg.Devices.Hostdevs {
		if hd.SubsysPCI == nil {
			continue
		}
		d := hd.SubsysPCI.Source.Address.Domain
		b := hd.SubsysPCI.Source.Address.Bus
		s := hd.SubsysPCI.Source.Address.Slot
		f := hd.SubsysPCI.Source.Address.Function
		addr := fmt.Sprintf("%04x:%02x:%02x.%01x", *d, *b, *s, *f)
		gpuAddrs = append(gpuAddrs, addr)
	}
	logger.Debugf(context.TODO(), "GPU addrs<%s>: %v", domcfg.Name, gpuAddrs)

	entry := &DomainCacheEntry{
		Name:     domcfg.Name,
		UUID:     domcfg.UUID,
		CPU:      cpu,
		Memory:   int64(memory),
		VNCPort:  vncPort,
		GPUAddrs: gpuAddrs,
		Schema:   domcfg,
		EruName:  types.EruID(domcfg.Name),
		IP:       meta.App.IP.IP,
		AppID:    meta.App.ID.ID,
		AppSID:   meta.App.ID.SID,
		AppName:  meta.App.Name.Name,
		UserName: meta.App.Owner.UserName,
		UserID:   meta.App.Owner.UserID,
	}
	return entry
}

func (vc *VMCache) updateAllDomainsHelper(l *libvirt.Libvirt) {
	logger := log.WithFunc("updateAllDomainsHelper")

	flags := libvirt.ConnectListDomainsActive | libvirt.ConnectListDomainsInactive
	domains, _, err := l.ConnectListAllDomains(1, flags)
	if err != nil {
		logger.Errorf(context.TODO(), err, "failed to list domains")
		return
	}

	newCache := map[string]*DomainCacheEntry{}
	for _, domain := range domains {
		var flags libvirt.DomainXMLFlags
		xmldoc, err := l.DomainGetXMLDesc(domain, flags)
		if err != nil {
			logger.Errorf(context.TODO(), err, "failed to get domain xml")
			return
		}

		domcfg := &libvirtxml.Domain{}
		if err = domcfg.Unmarshal(xmldoc); err != nil {
			logger.Errorf(context.TODO(), err, "failed to unmarshal domain xml")
			return
		}
		entry := extractInfoFromXML(domcfg)
		state, _, _ := l.DomainGetState(domain, 0)
		entry.State = libvirt.DomainState(state)
		newCache[entry.Name] = entry
	}

	// lock after libvirt operation, so it doesn't lock too long in this function
	vc.mu.Lock()
	defer vc.mu.Unlock()

	logger.Debugf(context.TODO(), "new cache: %v", newCache)
	vc.localDomainCache = newCache
}

func (vc *VMCache) updateAllDomains(ctx context.Context) {
	logger := log.WithFunc("updateAllDomains")
	defer logger.Infof(ctx, "[updateAllDomains] exit")
	var (
		l   *libvirt.Libvirt
		err error
	)
	firstCh := make(chan struct{}, 1)
	firstCh <- struct{}{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(90 * time.Second):
		case <-firstCh:
		}

		if l == nil || (!l.IsConnected()) {
			l, err = newLibvirt()
			if err != nil {
				logger.Errorf(ctx, err, "failed to get libvirt")
				continue
			}
		}
		vc.updateAllDomainsHelper(l)
	}
}

func (vc *VMCache) updateOneDomain(ctx context.Context, l *libvirt.Libvirt, dom libvirt.Domain) error {
	var flags libvirt.DomainXMLFlags
	xmldoc, err := l.DomainGetXMLDesc(dom, flags)
	if err != nil {
		return err
	}

	domcfg := &libvirtxml.Domain{}
	if err = domcfg.Unmarshal(xmldoc); err != nil {
		return err
	}
	entry := extractInfoFromXML(domcfg)
	state, _, _ := l.DomainGetState(dom, 0)
	entry.State = libvirt.DomainState(state)

	vc.mu.Lock()
	defer vc.mu.Unlock()
	log.WithFunc("updateOneDomain").Debugf(ctx, "new entry: %s, %v", entry.Name, entry.State)
	vc.localDomainCache[entry.Name] = entry
	return nil
}

func (vc *VMCache) processLibvirtEvents(ctx context.Context, l *libvirt.Libvirt, ch <-chan libvirt.DomainEventLifecycleMsg) {
	logger := log.WithFunc("processLibvirtEvents")
	for {
		select {
		case evt := <-ch:
			if evt.Dom.Name == "" {
				logger.Warnf(ctx, "event channel seems closed %v", evt)
				return
			}
			logger.Infof(ctx, "got event %v", evt)
			switch evt.Event {
			case DomainEventUndefined:
				vc.mu.Lock()
				logger.Infof(ctx, "delete domain %s", evt.Dom.Name)
				delete(vc.localDomainCache, evt.Dom.Name)
				vc.mu.Unlock()
			default:
				if err := vc.updateOneDomain(ctx, l, evt.Dom); err != nil {
					logger.Errorf(ctx, err, "failed to update domain %s", evt.Dom.Name)
				}
			}
			vc.NotifyEvent(evt)
		case <-ctx.Done():
			logger.Infof(context.TODO(), "[processLibvirtEvents] ctx done")
			return
		}
	}
}

func (vc *VMCache) handleDomainEvents(ctx context.Context) {
	logger := log.WithFunc("handleDomainEvents")
	defer logger.Infof(ctx, "[handleDomainEvents] exit")
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}

		l, err := newLibvirt()
		if err != nil {
			logger.Errorf(ctx, err, "failed to get libvirt")
			continue
		}

		// we should register events listener before fetching all existing domains
		evtCtx, evtCancel := context.WithCancel(ctx)
		ch, err := l.LifecycleEvents(evtCtx)
		if err != nil {
			logger.Errorf(ctx, err, "failed to get lifecycle events")
			evtCancel()
			continue
		}
		vc.processLibvirtEvents(ctx, l, ch)
		evtCancel()
	}
}

func (vc *VMCache) updateStats(ctx context.Context) {
	logger := log.WithFunc("updateStats")
	var (
		l   *libvirt.Libvirt
		err error
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(vc.statsUpdateInterval):
		}

		if l == nil || (!l.IsConnected()) {
			l, err = newLibvirt()
			if err != nil {
				logger.Errorf(ctx, err, "failed to get libvirt")
				continue
			}
		}

		flags := libvirt.ConnectGetAllDomainsStatsRunning
		statList, err := l.ConnectGetAllDomainStats(nil, 0, flags)
		if err != nil {
			logger.Errorf(ctx, err, "failed to get all domain stats")
			continue
		}
		for _, stats := range statList {
			statsMap := make(map[string]libvirt.TypedParam)
			for _, p := range stats.Params {
				statsMap[p.Field] = p
			}
			vc.statsCache.Set(stats.Dom.Name, statsMap, cache.DefaultExpiration)
		}
	}
}

func (vc *VMCache) Run(ctx context.Context, enableMetrics bool) {
	go vc.updateAllDomains(ctx)
	go vc.handleDomainEvents(ctx)
	if enableMetrics {
		go vc.updateStats(ctx)
	}
}

func FetchStats() map[string]DomainStatsResp {
	resps := make(map[string]DomainStatsResp)
	gVC.mu.Lock()
	defer gVC.mu.Unlock()
	for name, entry := range gVC.localDomainCache {
		if stats, ok := gVC.statsCache.Get(name); ok {
			resp := DomainStatsResp{
				DomainCacheEntry: *entry,
				Stats:            stats.(map[string]libvirt.TypedParam),
			}
			resps[name] = resp
		}
	}
	return resps
}

func FetchDomainEntry(name string) *DomainCacheEntry {
	gVC.mu.Lock()
	defer gVC.mu.Unlock()
	return gVC.localDomainCache[name]
}

func FetchGPUAddrs() []string {
	gVC.mu.Lock()
	defer gVC.mu.Unlock()

	resps := make([]string, 0)
	for _, entry := range gVC.localDomainCache {
		resps = append(resps, entry.GPUAddrs...)
	}
	return resps
}

func FetchDomainGPUAddrs() map[string][]string {
	gVC.mu.Lock()
	defer gVC.mu.Unlock()

	resps := make(map[string][]string, 0)
	for name, entry := range gVC.localDomainCache {
		resps[name] = entry.GPUAddrs
	}
	return resps
}

func FetchDomainsInfo() map[string]any {
	gVC.mu.Lock()
	defer gVC.mu.Unlock()

	resps := make(map[string]any)
	for name, entry := range gVC.localDomainCache {
		resps[name] = map[string]any{
			"gpus":  entry.GPUAddrs,
			"state": State2Str(entry.State),
		}
	}
	return resps
}

// UpdateDomain is used to update a domain in cache immediately
func UpdateDomain(name string) error {
	l, err := newLibvirt()
	if err != nil {
		return err
	}
	defer l.Disconnect() //nolint
	dom, err := l.DomainLookupByName(name)
	if err != nil {
		return err
	}
	return gVC.updateOneDomain(context.TODO(), l, dom)
}

func Setup(ctx context.Context, cfg *configs.Config, ws *interutils.Watchers) error {
	statsUpdateInterval := 10 * time.Second
	gVC = &VMCache{
		localDomainCache:    make(map[string]*DomainCacheEntry),
		statsCache:          cache.New(statsUpdateInterval+time.Second, statsUpdateInterval),
		statsUpdateInterval: 10 * time.Second,
		watchers:            ws,
	}
	gVC.Run(ctx, cfg.EnableLibvirtMetrics)
	return nil
}
