package host

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/model"
	"github.com/projecteru2/yavirt/vnet"
)

func addFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name: "cpu",
		},
		&cli.Int64Flag{
			Name: "memory",
		},
		&cli.Int64Flag{
			Name: "storage",
		},
		&cli.StringFlag{
			Name: "subnet",
		},
		&cli.StringFlag{
			Name: "network",
		},
	}
}

func add(c *cli.Context, runtime run.Runtime) error {
	hn := c.Args().First()
	subnet := runtime.ConvDecimal(c.String("subnet"))
	cpu := c.Int("cpu")
	mem := c.Int64("memory")
	storage := c.Int64("storage")
	network := c.String("network")

	switch {
	case len(hn) < 1:
		return errors.New("host name is required")
	case cpu < 1:
		return errors.New("--cpu is required")
	case mem < 1:
		return errors.New("--memory is required")
	case storage < 1:
		return errors.New("--storage is required")
	case network == vnet.NetworkVlan && subnet < 1:
		return errors.New("--subnet is required")
	case network != vnet.NetworkCalico && network != vnet.NetworkVlan:
		return errors.Errorf("--network is invalid: %s", network)
	}

	host := model.NewHost()
	host.Name = hn
	host.Type = model.HostVirtType
	host.Subnet = subnet
	host.CPU = cpu
	host.Memory = mem
	host.Storage = storage
	host.NetworkMode = network

	if err := host.Create(); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("host (%d) %s created\n", host.ID, host.Name)

	return nil
}
