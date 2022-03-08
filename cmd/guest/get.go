package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/errors"
)

func get(c *cli.Context, runtime run.Runtime) error {
	id := c.Args().First()
	if len(id) < 1 {
		return errors.New("Guest ID is required")
	}

	g, err := runtime.Guest.Load(runtime.VirtContext(), id)
	if err != nil {
		return err
	}
	fmt.Printf("guest: %s\n", g.ID)
	fmt.Printf("CPU: %d\n", g.CPU)
	fmt.Printf("Memory: %d\n", g.Memory)

	fmt.Println("volume:")
	for _, vol := range g.Vols {
		fmt.Printf("  %s\n", vol)
	}

	fmt.Println("IP:")
	for _, ip := range g.IPs {
		fmt.Printf("  %s, gw: %s\n", ip, ip.GatewayAddr())
	}

	hc, err := g.HealthCheck()
	if err != nil {
		if errors.Contain(err, errors.ErrKeyNotExists) {
			return nil
		}
		return err
	}
	fmt.Println("HealthCheck:")
	fmt.Printf("  %v\n", hc.TCPEndpoints())
	fmt.Printf("  %v\n", hc.HTTPEndpoints())

	return nil
}
