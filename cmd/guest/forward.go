package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func forwardFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "status",
			Required: true,
		},
	}
}

func forward(c *cli.Context, runtime run.Runtime) error {
	validStatus := func(st string) error {
		for _, status := range model.AllStatuses {
			if st == status {
				return nil
			}
		}
		return errors.Errorf("invalid dest. status: %s", st)
	}

	st := c.String("status")
	if err := validStatus(st); err != nil {
		return err
	}

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	g, err := runtime.Guest.Load(runtime.VirtContext(), id)
	if err != nil {
		return err
	}

	if err := g.ForwardStatus(st, false); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s forward to %s\n", id, st)

	return nil
}
