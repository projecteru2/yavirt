package maint

import (
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "maint",
		Subcommands: []*cli.Command{
			{
				Name:   "fasten",
				Action: run.Run(fasten),
			},
		},
	}
}
