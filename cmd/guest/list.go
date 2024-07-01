package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

func listFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "node",
		},
		&cli.BoolFlag{
			Name: "all",
		},
	}
}

func listCmd(c *cli.Context, _ run.Runtime) error {
	all := c.Bool("all")

	var err error
	var guests []*models.Guest
	if all {
		guests, err = models.GetAllGuests()
	} else {
		nodename := c.String("node")
		if len(nodename) < 1 {
			nodename = configs.Hostname()
		}
		guests, err = models.GetNodeGuests(nodename)
	}
	if err != nil && !errors.Is(err, terrors.ErrKeyNotExists) {
		return err
	}

	for _, g := range guests {
		fmt.Printf("%s\n", g.ID)
	}

	return nil
}
