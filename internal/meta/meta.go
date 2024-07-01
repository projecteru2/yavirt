package meta

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/store"
)

// Create .
func Create(res Resources) error {
	var data, err = res.Encode()
	if err != nil {
		return errors.Wrap(err, "failed to encode resources")
	}

	var ctx, cancel = Context(context.Background())
	defer cancel()

	if err := store.Create(ctx, data); err != nil {
		return errors.Wrap(err, "failed to create resources")
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
		return errors.Wrap(err, "failed to load resource")
	}

	res.SetVer(ver)

	return nil
}

func LoadRaw(key string) (map[string]any, int64, error) {
	var ctx, cancel = Context(context.Background())
	defer cancel()
	val := map[string]any{}
	var ver, err = store.Get(ctx, key, &val)
	if err != nil {
		return nil, ver, errors.Wrap(err, "failed to load resource")
	}

	return val, ver, nil
}

// Save .
func Save(res Resources) error {
	var data, err = res.Encode()
	if err != nil {
		return errors.Wrap(err, "failed to encode resources")
	}

	var ctx, cancel = Context(context.Background())
	defer cancel()

	if err := store.Update(ctx, data, res.Vers()); err != nil {
		return errors.Wrap(err, "failed to update resources")
	}

	res.IncrVer()

	return nil
}

// Context .
func Context(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, configs.Conf.MetaTimeout)
}
