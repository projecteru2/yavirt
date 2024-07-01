package boar

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/robfig/cron/v3"
)

// ListSnapshot .
func (svc *Boar) ListSnapshot(ctx context.Context, req types.ListSnapshotReq) (snaps types.Snapshots, err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, req.ID)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	volSnap, err := g.ListSnapshot(req.VolID)

	for vol, s := range volSnap {
		for _, snap := range s {
			snaps = append(snaps, &types.Snapshot{
				VolID:       vol.GetID(),
				VolMountDir: vol.GetMountDir(),
				SnapID:      snap.GetID(),
				CreatedTime: snap.GetCreatedTime(),
			})
		}
	}

	return
}

// CreateSnapshot .
func (svc *Boar) CreateSnapshot(ctx context.Context, req types.CreateSnapshotReq) (err error) {
	defer logErr(err)
	volID := req.VolID

	return svc.ctrl(ctx, req.ID, intertypes.CreateSnapshotOp, func(g *guest.Guest) error {
		suspended := false
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Suspend(); err != nil {
				return err
			}
			suspended = true
		}

		if err := g.CreateSnapshot(volID); err != nil {
			return err
		}

		if err := g.CheckVolume(volID); err != nil {

			if suspended {
				if err := g.Stop(ctx, true); err != nil {
					return err
				}
				suspended = false
				stopped = true
			}

			if err := g.RepairVolume(volID); err != nil {
				return err
			}
		}

		if suspended {
			return g.Resume()
		} else if stopped {
			return g.Start(ctx, false)
		}
		return nil
	}, nil)
}

// CommitSnapshot .
func (svc *Boar) CommitSnapshot(ctx context.Context, req types.CommitSnapshotReq) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, req.ID, intertypes.CommitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshot(req.VolID, req.SnapID); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx, false)
		}
		return nil
	}, nil)
}

// CommitSnapshotByDay .
func (svc *Boar) CommitSnapshotByDay(ctx context.Context, id, volID string, day int) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, id, intertypes.CommitSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.CommitSnapshotByDay(volID, day); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx, false)
		}
		return nil
	}, nil)
}

// RestoreSnapshot .
func (svc *Boar) RestoreSnapshot(ctx context.Context, req types.RestoreSnapshotReq) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, req.ID, intertypes.RestoreSnapshotOp, func(g *guest.Guest) error {
		stopped := false
		if g.Status == meta.StatusRunning {
			if err := g.Stop(ctx, true); err != nil {
				return err
			}
			stopped = true
		}

		if err := g.RestoreSnapshot(req.VolID, req.SnapID); err != nil {
			return err
		}

		if stopped {
			return g.Start(ctx, false)
		}
		return nil
	}, nil)
}

// TODO: Decide time
func (svc *Boar) ScheduleSnapshotCreate() error {
	c := cron.New()

	// Everyday 3am
	if _, err := c.AddFunc("0 3 * * *", svc.batchCreateSnapshot); err != nil {
		return errors.Wrap(err, "")
	}

	// Every Sunday 1am
	if _, err := c.AddFunc("0 1 * * SUN", svc.batchCommitSnapshot); err != nil {
		return errors.Wrap(err, "")
	}

	// Start job asynchronously
	c.Start()

	return nil
}

func (svc *Boar) batchCreateSnapshot() {
	logger := log.WithFunc("Boar.batchCreateSnapshot")
	guests, err := models.GetAllGuests()
	if err != nil {
		logger.Error(context.TODO(), err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			req := types.CreateSnapshotReq{
				ID:    g.ID,
				VolID: volID,
			}

			if err := svc.CreateSnapshot(context.TODO(), req); err != nil {
				logger.Error(context.TODO(), err)
				metrics.IncrError()
			}
		}
	}
}

func (svc *Boar) batchCommitSnapshot() {
	logger := log.WithFunc("Boar.batchCommitSnapshot")
	guests, err := models.GetAllGuests()
	if err != nil {
		logger.Error(context.TODO(), err)
		metrics.IncrError()
		return
	}

	for _, g := range guests {
		for _, volID := range g.VolIDs {
			if err := svc.CommitSnapshotByDay(
				context.TODO(),
				g.ID,
				volID,
				configs.Conf.SnapshotRestorableDay,
			); err != nil {
				logger.Error(context.TODO(), err)
				metrics.IncrError()
			}
		}
	}
}
