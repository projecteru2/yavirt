package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/ctl/run"
	"github.com/projecteru2/yavirt/errors"
)

func connectExtraNetworkFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "network",
			Required: true,
		},
		&cli.StringFlag{
			Name: "ipv4",
		},
	}
}

func disconnectExtraNetworkFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "network",
			Required: true,
		},
	}
}

func disconnectExtraNetwork(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	network := c.String("network")

	if err := runtime.Guest.DisconnectExtraNetwork(runtime.VirtContext(), id, network); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("guest %s had been disconnected from network %s\n", id, network)

	return nil
}

func connectExtraNetwork(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return fmt.Errorf("Guest ID is required")
	}

	network := c.String("network")
	ipv4 := c.String("ipv4")

	dest, err := runtime.Guest.ConnectExtraNetwork(runtime.VirtContext(), id, network, ipv4)
	if err != nil {
		return errors.Trace(err)
	}

	if len(ipv4) < 1 {
		fmt.Printf("assigned network %s IP %s for guest %s\n", network, dest, id)
	} else {
		fmt.Printf("bound network %s IP %s for guest %s\n", network, dest, id)
	}

	return nil
}
