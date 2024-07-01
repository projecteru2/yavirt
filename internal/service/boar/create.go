package boar

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/pkg/utils"

	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/guest"
)

// CreateGuest .
func (svc *Boar) CreateGuest(ctx context.Context, opts intertypes.GuestCreateOption) (*types.Guest, error) {
	logger := log.WithFunc("boar.CreateGuest")
	if opts.CPU == 0 {
		opts.CPU = utils.Min(svc.Host.CPU, configs.Conf.Resource.MaxCPU)
	}
	if opts.Mem == 0 {
		opts.Mem = utils.Min(svc.Host.Memory, configs.Conf.Resource.MaxMemory)
	}
	ctx = interutils.NewRollbackListContext(ctx)
	g, err := svc.Create(ctx, opts, svc.Host)
	if err != nil {
		logger.Error(ctx, err)
		metrics.IncrError()
		rl := interutils.GetRollbackListFromContext(ctx)
		for {
			fn, msg := rl.Pop()
			if fn == nil {
				break
			}
			logger.Infof(ctx, "start to rollback<%s>", msg)
			if err := fn(); err != nil {
				log.Errorf(ctx, err, "failed to rollback<%s>", msg)
			}
		}
		return nil, err
	}

	go func() {
		svc.BootGuestCh <- g.ID
	}()

	return convGuestResp(g.Guest), nil
}

// Create creates a new guest.
func (svc *Boar) Create(ctx context.Context, opts intertypes.GuestCreateOption, host *models.Host) (*guest.Guest, error) {
	logger := log.WithFunc("boar.Create")
	vols, err := extractVols(opts.Resources)
	if err != nil {
		return nil, err
	}

	// Creates metadata.
	g, err := models.CreateGuest(opts, host, vols)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	rl := interutils.GetRollbackListFromContext(ctx)

	// session locker is locked here and is released in start
	lck := interutils.NewCreateSessionFlock(g.ID)
	if err := lck.Trylock(); err != nil {
		logger.Warnf(ctx, "failed to lock create seesion id<%s> %s", g.ID, err)
	} else {
		rl.Append(func() error { return lck.RemoveFile() }, "release creation session locker")
	}

	rl.Append(func() error { return g.Delete(true) }, "Delete guest model")

	logger.Debugf(ctx, "Guest Created: %+v", g)
	// Destroys resource and delete metadata while rolling back.
	vg := guest.New(ctx, g)

	// Creates the resource.
	create := func(ctx context.Context) (any, error) {
		err := svc.create(ctx, vg)
		return nil, err
	}

	_, err = svc.do(ctx, g.ID, intertypes.CreateOp, create, nil)
	return vg, err
}

func (svc *Boar) create(ctx context.Context, vg *guest.Guest) (err error) {
	logger := log.WithFunc("Boar.create").WithField("guest", vg.ID)
	logger.Debugf(ctx, "starting to cache image")
	if err := vg.CacheImage(&svc.imageMutex); err != nil {
		return errors.Wrap(err, "")
	}

	logger.Debug(ctx, "creating network")
	if err = vg.CreateNetwork(ctx); err != nil {
		return err
	}
	logger.Debug(ctx, "preparing volumes")
	if err = vg.PrepareVolumesForCreate(ctx); err != nil {
		return err
	}
	logger.Debug(ctx, "defining guest")
	if err = vg.DefineGuestForCreate(ctx); err != nil {
		return errors.Wrap(err, "")
	}

	return nil
}
