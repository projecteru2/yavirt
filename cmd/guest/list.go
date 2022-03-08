package guest

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/model"
	"github.com/projecteru2/yavirt/util"
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

func list(c *cli.Context, runtime run.Runtime) error {
	all := c.Bool("all")

	var err error
	var guests []*model.Guest
	if all {
		guests, err = model.GetAllGuests()
	} else {
		nodename := c.String("node")
		if len(nodename) < 1 {
			nodename, err = util.Hostname()
			if err != nil {
				return err
			}
		}
		guests, err = model.GetNodeGuests(nodename)
	}
	if err != nil {
		return err
	}

	for _, g := range guests {
		fmt.Printf("%s\n", g.ID)
	}

	return nil
}
