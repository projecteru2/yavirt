package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/local"
)

func resizeFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name: "volumes",
		},
		&cli.IntFlag{
			Name: "cpu",
		},
		&cli.Int64Flag{
			Name: "memory",
		},
	}
}

func resize(c *cli.Context, runtime run.Runtime) (err error) {
	vs := map[string]volume.Volume{}
	for _, raw := range c.StringSlice("volumes") {
		vol, err := local.NewVolumeFromStr(raw)
		if err != nil {
			return errors.Wrap(err, "")
		}
		vs[vol.GetMountDir()] = vol
	}

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	cpu := c.Int("cpu")
	mem := c.Int64("memory")
	req := &types.GuestResizeOption{
		ID:  id,
		CPU: cpu,
		Mem: mem,
		//TODO: add resources
	}
	if err = runtime.Svc.ResizeGuest(runtime.Ctx, id, req); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s resized\n", id)

	return
}
