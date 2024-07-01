package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
)

func captureFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
		},
		&cli.BoolFlag{
			Name: "overridden",
		},
	}
}

func capture(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	name := c.String("name")
	overridden := c.Bool("overridden")
	_, err := runtime.Svc.CaptureGuest(runtime.Ctx, id, name, overridden)
	if err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s captured\n", name)

	return nil
}
