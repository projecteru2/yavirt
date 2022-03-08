package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func listSnapshotFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name: "all",
		},
		&cli.StringFlag{
			Name: "vol",
		},
	}
}

func createSnapshotFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "vol",
		},
	}
}

func commitSnapshotFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "vol",
		},
		&cli.StringFlag{
			Name: "snap",
		},
		&cli.IntFlag{
			Name: "day",
		},
	}
}

func restoreSnapshotFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "vol",
		},
		&cli.StringFlag{
			Name: "snap",
		},
	}
}

func listSnapshot(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	all := c.Bool("all")
	volID := ""
	if !all {
		volID = c.String("vol")
		if len(volID) < 1 {
			return errors.New("Either --all or vol id is required")
		}
	}

	volSnap, err := runtime.Guest.ListSnapshot(runtime.VirtContext(), id, volID)
	if err != nil {
		return errors.Trace(err)
	}

	for vol, snaps := range volSnap {
		fmt.Printf("Vol: %s\n", vol)
		fmt.Printf("Total: %d snapshot(s)\n", len(snaps))
		for _, s := range snaps {
			fmt.Printf("%s\n", s)
		}
		fmt.Println()
	}

	return nil
}

func createSnapshot(c *cli.Context, runtime run.Runtime) error {

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	volID := c.String("vol")
	if len(volID) < 1 {
		return errors.New("Volume ID is required")
	}

	return runtime.Guest.CreateSnapshot(runtime.VirtContext(), id, volID)
}

func commitSnapshot(c *cli.Context, runtime run.Runtime) error {

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	volID := c.String("vol")
	if len(volID) < 1 {
		return errors.New("Volume ID is required")
	}

	snapID := c.String("snap")
	day := c.Int("day")
	if len(snapID) < 1 && day <= 0 {
		return errors.New("Either Snapshot ID or positive day is required")
	} else if len(snapID) > 0 && day > 0 {
		return errors.New("Can only specify one of either Snapshot ID or day")
	}

	if len(snapID) > 0 {
		return runtime.Guest.CommitSnapshot(runtime.VirtContext(), id, volID, snapID)
	}
	return runtime.Guest.CommitSnapshotByDay(runtime.VirtContext(), id, volID, day)

}

func restoreSnapshot(c *cli.Context, runtime run.Runtime) error {

	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	volID := c.String("vol")
	if len(volID) < 1 {
		return errors.New("Volume ID is required")
	}

	snapID := c.String("snap")
	if len(snapID) < 1 {
		return errors.New("Snapshot ID is required")
	}

	return runtime.Guest.RestoreSnapshot(runtime.VirtContext(), id, volID, snapID)
}
