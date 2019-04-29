package host

import (
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/ctl/run"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "host",
		Subcommands: []*cli.Command{
			{
				Name:   "add",
				Flags:  addFlags(),
				Action: run.Run(add),
			},
			{
				Name:   "get",
				Action: run.Run(get),
			},
		},
	}
}
