package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
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
	vs := map[string]int64{}
	for _, raw := range c.StringSlice("volumes") {
		mnt, cap := utils.PartRight(raw, ":") //nolint
		if vs[mnt], err = utils.Atoi64(cap); err != nil {
			return errors.Trace(err)
		}
	}

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	cpu := c.Int("cpu")
	mem := c.Int64("memory")
	if err = runtime.Guest.Resize(runtime.VirtContext(), id, cpu, mem, vs); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s resized\n", id)

	return
}
