package guest

import (
	"os"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/urfave/cli/v2"
)

func execFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "i",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "devname",
			Value: "",
		},
		&cli.BoolFlag{
			Name:  "force",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "safe",
			Value: false,
		},
	}
}

func exec(c *cli.Context, runtime run.Runtime) (err error) {
	// defer runtime.CancelFn()

	if c.Bool("i") {
		return attachGuest(c, runtime)
	} else { //nolint
		return execGuest(c, runtime)
	}
}

func execGuest(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	cmds := c.Args().Tail()

	log.Debugf(c.Context, "exec guest %s, cmd: %v", id, cmds)
	msg, err := runtime.Svc.ExecuteGuest(runtime.Ctx, id, cmds)
	if err != nil {
		log.Errorf(c.Context, err, "exec guest error")
		return err
	}
	if msg.ExitCode == 0 {
		os.Stdout.Write(msg.Data)
	} else {
		os.Stderr.Write(msg.Data)
	}
	log.Debugf(c.Context, "+_+_+_   %s", string(msg.Data))
	return err
}
