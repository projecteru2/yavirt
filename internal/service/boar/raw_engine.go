package boar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/meta"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/vmcache"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	vmiFact "github.com/yuyang0/vmimage/factory"
)

type VMParams struct {
	DeviceName string `json:"device_name"`
}

func (svc *Boar) RawEngine(ctx context.Context, id string, req types.RawEngineReq) (types.RawEngineResp, error) {
	switch req.Op {
	case "vm-get-vnc-port":
		return svc.getVNCPort(ctx, id)
	case "vm-init-sys-disk":
		return svc.InitSysDisk(ctx, id, req.Params)
	case "vm-fs-freeze-all":
		return svc.fsFreezeAll(ctx, id)
	case "vm-fs-thaw-all":
		return svc.fsThawAll(ctx, id)
	case "vm-list-vols", "vm-list-vol", "vm-list-volume", "vm-list-volumes":
		return svc.listVolumes(ctx, id)
	case "vm-fs-freeze-status":
		return svc.fsFreezeStatus(ctx, id)
	default:
		return types.RawEngineResp{}, errors.Errorf("invalid operation %s", req.Op)
	}
}

func (svc *Boar) getVNCPort(_ context.Context, id string) (types.RawEngineResp, error) {
	entry := vmcache.FetchDomainEntry(id)
	var port int
	if entry != nil {
		port = entry.VNCPort
	}
	obj := map[string]int{
		"port": port,
	}
	bs, _ := json.Marshal(obj)
	resp := types.RawEngineResp{
		Data: bs,
	}
	return resp, nil
}

type VolItem struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Device string `json:"device"`
}

func guestVolumes2VolItemList(vols volFact.Volumes) []VolItem {
	if len(vols) == 0 {
		return nil
	}
	volItemList := make([]VolItem, 0)
	for _, item := range vols {
		volItem := VolItem{
			Name:   item.Name(),
			Size:   item.GetSize(),
			Device: item.GetDevice(),
		}
		volItemList = append(volItemList, volItem)
	}
	return volItemList
}

func (svc *Boar) listVolumes(ctx context.Context, id string) (types.RawEngineResp, error) {
	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}

	volItemList := guestVolumes2VolItemList(g.Vols)
	if volItemList == nil {
		return types.RawEngineResp{}, nil
	}

	bs, _ := json.Marshal(volItemList)
	resp := types.RawEngineResp{
		Data: bs,
	}
	return resp, nil
}

func (svc *Boar) fsFreezeAll(ctx context.Context, id string) (types.RawEngineResp, error) {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if nFS, err := g.FSFreezeAll(ctx); err != nil {
			return nil, errors.Wrap(err, "")
		} else { //nolint
			return nFS, nil
		}
	}
	nFSRaw, err := svc.do(ctx, id, intertypes.FSThawOP, do, nil)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}
	nFS, _ := nFSRaw.(int)
	return types.RawEngineResp{
		Data: []byte(fmt.Sprintf(`{"fs_count": %d}`, nFS)),
	}, nil
}

func (svc *Boar) fsThawAll(ctx context.Context, id string) (types.RawEngineResp, error) {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if nFS, err := g.FSThawAll(ctx); err != nil {
			return nil, errors.Wrap(err, "")
		} else { //nolint
			return nFS, nil
		}
	}
	nFSRaw, err := svc.do(ctx, id, intertypes.FSThawOP, do, nil)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}
	nFS, _ := nFSRaw.(int)
	return types.RawEngineResp{
		Data: []byte(fmt.Sprintf(`{"fs_count": %d}`, nFS)),
	}, nil
}

func (svc *Boar) fsFreezeStatus(ctx context.Context, id string) (types.RawEngineResp, error) {
	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}
	status, err := g.FSFreezeStatus(ctx)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}
	return types.RawEngineResp{
		Data: []byte(fmt.Sprintf(`{"status": "%s"}`, status)),
	}, nil
}

func (svc *Boar) InitSysDisk(ctx context.Context, id string, rawParams []byte) (types.RawEngineResp, error) {
	logger := log.WithFunc("boar.InitSysDisk")
	args := &intertypes.InitSysDiskArgs{}
	if err := json.Unmarshal(rawParams, args); err != nil {
		return types.RawEngineResp{}, errors.Wrapf(err, "failed to unmarshal params")
	}
	logger.Infof(ctx, "[InitSysDisk] params: %v", args)
	// prepare image
	img, err := vmiFact.LoadImage(ctx, args.Image)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrapf(err, "failed to load image %s", args.Image)
	}
	vols, err := extractVols(args.Resources)
	if err != nil {
		return types.RawEngineResp{}, errors.Wrapf(err, "failed to extract new sys volume")
	}
	if len(vols) != 1 || (!vols[0].IsSys()) {
		return types.RawEngineResp{}, errors.Wrapf(err, "need a new sys volume, but gives %v", vols[0])
	}
	newSysVol := vols[0]
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if g.Status != meta.StatusStopped {
			return nil, errors.Newf("guest should in stopped state")
		}

		if err := g.InitSysDisk(ctx, img, args, newSysVol); err != nil {
			return nil, errors.Wrap(err, "")
		}
		if err := g.Start(ctx, true); err != nil {
			return nil, errors.Wrapf(err, "failed to start guest %s", g.ID)
		}

		return nil, nil //nolint
	}
	if _, err := svc.do(ctx, id, intertypes.StopOp, do, nil); err != nil {
		return types.RawEngineResp{}, errors.Wrap(err, "")
	}
	msg := `{"success":true}`
	resp := types.RawEngineResp{
		Data: []byte(msg),
	}
	return resp, nil
}
