package calico

import (
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "calico",
		Subcommands: []*cli.Command{
			{
				Name:   "align",
				Flags:  alignFlags(),
				Action: run.Run(align),
			},
		},
	}
}
