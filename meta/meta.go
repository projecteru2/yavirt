package meta

import (
	"context"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/store"
)

// Create .
func Create(res Resources) error {
	var data, err = res.Encode()
	if err != nil {
		return errors.Trace(err)
	}

	var ctx, cancel = Context(context.Background())
	defer cancel()

	if err := store.Create(ctx, data); err != nil {
		return errors.Trace(err)
	}

	res.IncrVer()

	return nil
}

// Load .
func Load(res Resource) error {
	var ctx, cancel = Context(context.Background())
	defer cancel()

	var ver, err = store.Get(ctx, res.MetaKey(), res)
	if err != nil {
		return errors.Trace(err)
	}

	res.SetVer(ver)

	return nil
}

// Save .
func Save(res Resources) error {
	var data, err = res.Encode()
	if err != nil {
		return errors.Trace(err)
	}

	var ctx, cancel = Context(context.Background())
	defer cancel()

	if err := store.Update(ctx, data, res.Vers()); err != nil {
		return errors.Trace(err)
	}

	res.IncrVer()

	return nil
}

// Context .
func Context(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, config.Conf.MetaTimeout.Duration())
}
