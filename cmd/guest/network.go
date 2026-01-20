package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
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

	if err := runtime.Svc.DisconnectNetwork(runtime.Ctx, id, network); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("guest %s had been disconnected from network %s\n", id, network)

	return nil
}

func connectExtraNetwork(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return fmt.Errorf("guest ID is required")
	}

	network := c.String("network")
	ipv4 := c.String("ipv4")

	dest, err := runtime.Svc.ConnectNetwork(runtime.Ctx, id, network, ipv4)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if len(ipv4) < 1 {
		fmt.Printf("assigned network %s IP %s for guest %s\n", network, dest, id)
	} else {
		fmt.Printf("bound network %s IP %s for guest %s\n", network, dest, id)
	}

	return nil
}
