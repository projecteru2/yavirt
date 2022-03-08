package guest

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/virt"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func destroyFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
	}
}

func stopFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
	}
}

func start(c *cli.Context, runtime run.Runtime) error {
	id, err := op(c, runtime, runtime.Guest.Start)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s started\n", id)

	return nil
}

func suspend(c *cli.Context, runtime run.Runtime) error {
	id, err := op(c, runtime, runtime.Guest.Suspend)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s suspended\n", id)

	return nil
}

func resume(c *cli.Context, runtime run.Runtime) error {
	id, err := op(c, runtime, runtime.Guest.Resume)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s resumed\n", id)

	return nil
}

func stop(c *cli.Context, runtime run.Runtime) error {
	shut := func(ctx virt.Context, id string) error {
		return runtime.Guest.Stop(ctx, id, c.Bool("force"))
	}

	id, err := op(c, runtime, shut)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s stopped\n", id)

	return nil
}

func destroy(c *cli.Context, runtime run.Runtime) error {
	destroy := func(ctx virt.Context, id string) error {
		done, err := runtime.Guest.Destroy(ctx, id, c.Bool("force"))
		if err != nil {
			return errors.Trace(err)
		}

		select {
		case err := <-done:
			return err
		case <-time.After(time.Minute):
			return errors.ErrTimeout
		}
	}

	id, err := op(c, runtime, destroy)
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s destroyed\n", id)

	return nil
}

func op(c *cli.Context, runtime run.Runtime, fn func(virt.Context, string) error) (id string, err error) {
	id = c.Args().First()
	err = fn(runtime.VirtContext(), id)
	return
}
