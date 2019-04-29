package guest

import (
	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/ctl/run"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "guest",
		Subcommands: []*cli.Command{
			{
				Name:   "get",
				Action: run.Run(get),
			},
			{
				Name:   "list",
				Flags:  listFlags(),
				Action: run.Run(list),
			},
			{
				Name:   "create",
				Flags:  createFlags(),
				Action: run.Run(create),
			},
			{
				Name:   "start",
				Action: run.Run(start),
			},
			{
				Name:   "suspend",
				Action: run.Run(suspend),
			},
			{
				Name:   "resume",
				Action: run.Run(resume),
			},
			{
				Name:   "stop",
				Flags:  stopFlags(),
				Action: run.Run(stop),
			},
			{
				Name:   "destroy",
				Flags:  destroyFlags(),
				Action: run.Run(destroy),
			},
			{
				Name:   "forward",
				Flags:  forwardFlags(),
				Action: run.Run(forward),
			},
			{
				Name:   "resize",
				Flags:  resizeFlags(),
				Action: run.Run(resize),
			},
			{
				Name:   "capture",
				Flags:  captureFlags(),
				Action: run.Run(capture),
			},
			{
				Name:   "connect",
				Flags:  connectExtraNetworkFlags(),
				Action: run.Run(connectExtraNetwork),
			},
			{
				Name:   "disconnect",
				Flags:  disconnectExtraNetworkFlags(),
				Action: run.Run(disconnectExtraNetwork),
			},
			{
				Name:   "list-snapshot",
				Flags:  listSnapshotFlags(),
				Action: run.Run(listSnapshot),
			},
			{
				Name:   "create-snapshot",
				Flags:  createSnapshotFlags(),
				Action: run.Run(createSnapshot),
			},
			{
				Name:   "commit-snapshot",
				Flags:  commitSnapshotFlags(),
				Action: run.Run(commitSnapshot),
			},
			{
				Name:   "restore-snapshot",
				Flags:  restoreSnapshotFlags(),
				Action: run.Run(restoreSnapshot),
			},
		},
	}
}
