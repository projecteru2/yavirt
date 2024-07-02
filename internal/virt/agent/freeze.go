package agent

import (
	"context"

	"github.com/cockroachdb/errors"
)

func (a *Agent) FSFreezeAll(ctx context.Context) (int, error) {
	nFS, err := a.qmp.FSFreezeAll(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return nFS, nil
}

func (a *Agent) FSThawAll(ctx context.Context) (int, error) {
	nFS, err := a.qmp.FSThawAll(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return nFS, nil
}

func (a *Agent) FSFreezeStatus(ctx context.Context) (string, error) {
	status, err := a.qmp.FSFreezeStatus(ctx)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return status, nil
}
