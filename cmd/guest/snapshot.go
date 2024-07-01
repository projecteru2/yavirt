package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/cmd/run"
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

	req := types.ListSnapshotReq{
		ID:    id,
		VolID: volID,
	}
	volSnap, err := runtime.Svc.ListSnapshot(runtime.Ctx, req)
	if err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("Total: %d snapshot(s)\n", len(volSnap))
	for _, snap := range volSnap {
		fmt.Printf("%v\n", snap)
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

	req := types.CreateSnapshotReq{
		ID:    id,
		VolID: volID,
	}
	return runtime.Svc.CreateSnapshot(runtime.Ctx, req)
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
		req := types.CommitSnapshotReq{
			ID:     id,
			VolID:  volID,
			SnapID: snapID,
		}
		return runtime.Svc.CommitSnapshot(runtime.Ctx, req)
	}
	return runtime.Svc.CommitSnapshotByDay(runtime.Ctx, id, volID, day)

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

	req := types.RestoreSnapshotReq{
		ID:     id,
		VolID:  volID,
		SnapID: snapID,
	}
	return runtime.Svc.RestoreSnapshot(runtime.Ctx, req)
}
