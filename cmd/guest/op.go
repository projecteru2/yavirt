package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/cmd/run"
)

func controlFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
	}
}

func start(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf(c.Context, "Starting guest %s", id)

	if err := runtime.Svc.ControlGuest(runtime.Ctx, id, types.OpStart, c.Bool("force")); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s started\n", id)

	return nil
}

func suspend(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf(c.Context, "Suspending guest %s", id)
	if err := runtime.Svc.ControlGuest(runtime.Ctx, id, types.OpSuspend, false); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s suspended\n", id)

	return nil
}

func resume(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf(c.Context, "Resuming guest %s", id)
	if err := runtime.Svc.ControlGuest(runtime.Ctx, id, types.OpResume, false); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s resumed\n", id)

	return nil
}

func stop(c *cli.Context, runtime run.Runtime) error {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf(c.Context, "Stopping guest %s", id)
	if err := runtime.Svc.ControlGuest(runtime.Ctx, id, types.OpStop, c.Bool("force")); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s stopped\n", id)

	return nil
}

func destroy(c *cli.Context, runtime run.Runtime) (err error) {
	defer runtime.CancelFn()

	id := c.Args().First()
	log.Debugf(c.Context, "Destroying guest %s", id)

	err = runtime.Svc.ControlGuest(runtime.Ctx, id, types.OpDestroy, c.Bool("force"))
	if err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s destroyed\n", id)

	return nil
}
