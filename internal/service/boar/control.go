package boar

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/models"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/utils"
)

// ControlGuest .
func (svc *Boar) ControlGuest(ctx context.Context, id, operation string, force bool) (err error) {
	var errCh <-chan error
	switch operation {
	case types.OpStart:
		err = svc.startGuest(ctx, id, force)
	case types.OpStop:
		err = svc.stopGuest(ctx, id, force)
	case types.OpDestroy:
		errCh, err = svc.destroyGuest(ctx, id, force)
		if err != nil {
			break
		}
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case err = <-errCh:
		}
	case types.OpSuspend:
		err = svc.suspendGuest(ctx, id)
	case types.OpResume:
		err = svc.resumeGuest(ctx, id)
	}

	if err != nil {
		log.WithFunc("boar.ControlGuest").Error(ctx, err)
		metrics.IncrError()
		return errors.Wrap(err, "")
	}

	return nil
}

// destroyGuest destroys a guest.
func (svc *Boar) destroyGuest(ctx context.Context, id string, force bool) (<-chan error, error) {
	var done <-chan error
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id, models.IgnoreLoadImageErrOption())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if done, err = g.Destroy(ctx, force); err != nil {
			return nil, errors.Wrap(err, "")
		}

		return nil, nil //nolint
	}
	_, err := svc.do(ctx, id, intertypes.DestroyOp, do, nil)
	return done, err
}

// stopGuest stops a guest.
func (svc *Boar) stopGuest(ctx context.Context, id string, force bool) error {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id, models.IgnoreLoadImageErrOption())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if err := g.Stop(ctx, force); err != nil {
			return nil, errors.Wrap(err, "")
		}

		return nil, nil //nolint
	}
	_, err := svc.do(ctx, id, intertypes.StopOp, do, nil)

	// eru agent only track start and die event,
	// so also send a die event here
	if err == nil {
		svc.watchers.Watched(intertypes.Event{
			ID:   id,
			Type: guestEventType,
			Op:   intertypes.DieOp,
			Time: time.Now().UTC(),
		})
	}
	return err
}

// startGuest boots a guest.
func (svc *Boar) startGuest(ctx context.Context, id string, force bool) error {
	logger := log.WithFunc("boar.startGuest")
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id, models.IgnoreLoadImageErrOption())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		// we need to release the creation session locker here
		lck := utils.NewCreateSessionFlock(g.ID)
		defer func() {
			logger.Debugf(ctx, "[session unlocker] %s", g.ID)
			if err := lck.RemoveFile(); err != nil {
				logger.Warnf(ctx, "failed to remove session locker file %s", err)
			}
		}()

		if err := g.Start(ctx, force); err != nil {
			return nil, errors.Wrap(err, "")
		}
		if g.LambdaOption != nil && !g.LambdaStdin {
			output, exitCode, pid, err := g.ExecuteCommand(ctx, g.LambdaOption.Cmd)
			if err != nil {
				return nil, errors.Wrap(err, "")
			}
			g.LambdaOption.CmdOutput = output
			g.LambdaOption.ExitCode = exitCode
			g.LambdaOption.Pid = pid

			if err = g.Save(); err != nil {
				return nil, errors.Wrap(err, "")
			}
		}
		return nil, nil //nolint
	}
	defer logger.Debugf(ctx, "exit startGuest")
	_, err := svc.do(ctx, id, intertypes.StartOp, do, nil)
	return err
}

// suspendGuest suspends a guest.
func (svc *Boar) suspendGuest(ctx context.Context, id string) error {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id, models.IgnoreLoadImageErrOption())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if err := g.Suspend(); err != nil {
			return nil, errors.Wrap(err, "")
		}
		return nil, nil //nolint
	}
	_, err := svc.do(ctx, id, intertypes.SuspendOp, do, nil)
	return err
}

// resumeGuest resumes a suspended guest.
func (svc *Boar) resumeGuest(ctx context.Context, id string) error {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id, models.IgnoreLoadImageErrOption())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if err := g.Resume(); err != nil {
			return nil, errors.Wrap(err, "")
		}
		return nil, nil //nolint
	}
	_, err := svc.do(ctx, id, intertypes.ResumeOp, do, nil)
	return err
}
