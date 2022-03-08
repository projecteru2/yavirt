package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/guest"
	"github.com/projecteru2/yavirt/cmd/host"
	"github.com/projecteru2/yavirt/cmd/image"
	"github.com/projecteru2/yavirt/cmd/maint"
	"github.com/projecteru2/yavirt/cmd/network"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/ver"
)

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(ver.Version())
	}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "config",
				Usage:    "config files",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "skip-setup-host",
				Value: false,
			},
		},

		Commands: []*cli.Command{
			guest.Command(),
			image.Command(),
			host.Command(),
			network.Command(),
			maint.Command(),
		},

		Version: "v",
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(errors.Stack(err))
	}
}
