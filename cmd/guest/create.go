package guest

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func createFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:  "count",
			Value: 1,
		},
		&cli.IntFlag{
			Name:  "cpu",
			Value: 1,
		},
		&cli.Int64Flag{
			Name:  "memory",
			Value: utils.GB,
		},
		&cli.StringFlag{
			Name:  "storage",
			Usage: "mount info. like, --storage /data0:53687091200",
		},
		&cli.StringFlag{
			Name: "network",
		},
		&cli.StringFlag{
			Name: "dmi",
		},
		&cli.StringFlag{
			Name: "image-user",
		},
	}
}

func create(c *cli.Context, runtime run.Runtime) error {
	vols, err := getVols(c.String("storage"))
	if err != nil {
		return errors.Trace(err)
	}

	opts := types.GuestCreateOption{
		CPU:       c.Int("cpu"),
		Mem:       c.Int64("memory"),
		ImageName: c.Args().First(),
		ImageUser: c.String("image-user"),
		DmiUUID:   c.String("dmi"),
	}

	cnt := c.Int("count")
	network := c.String("network")

	if len(network) < 1 {
		network = runtime.Host.NetworkMode
	}

	switch {
	case len(opts.ImageName) < 1:
		return fmt.Errorf("image name is required")
	case opts.CPU < 1:
		return fmt.Errorf("--cpu is required")
	case opts.Mem < 1:
		return fmt.Errorf("--memory is required")
	case cnt < 1:
		return fmt.Errorf("--count must be greater than 0")
	case network != vnet.NetworkCalico && network != vnet.NetworkVlan:
		return fmt.Errorf("--network is invalid: %s", network)
	}

	runtime.Host.NetworkMode = network

	for i := 0; i < cnt; i++ {
		g, err := runtime.Guest.Create(runtime.VirtContext(), opts, runtime.Host, vols)
		if err != nil {
			return err
		}

		fmt.Printf("guest %s created\n\n", g.ID)
	}

	return nil
}

func getVols(mounts string) ([]*models.Volume, error) {
	if len(mounts) < 1 {
		return nil, nil
	}

	var vols = []*models.Volume{}

	for _, raw := range strings.Split(mounts, ",") {
		mnt, rawCap := utils.PartRight(raw, ":")

		volCap, err := utils.Atoi64(rawCap)
		if err != nil {
			return nil, errors.Trace(err)
		}

		vol, err := models.NewDataVolume(mnt, volCap)
		if err != nil {
			return nil, errors.Trace(err)
		}

		vols = append(vols, vol)
	}

	return vols, nil
}
