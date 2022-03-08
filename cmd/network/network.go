package network

import (
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/network/calico"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "network",
		Subcommands: []*cli.Command{
			calico.Command(),
		},
	}
}
