package host

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/errors"
)

func get(c *cli.Context, _ run.Runtime) error {
	hn := c.Args().First()
	if len(hn) < 1 {
		return errors.New("host name is required")
	}

	h, err := models.LoadHost(hn)
	if err != nil {
		return errors.Trace(err)

	}

	fmt.Println(h)

	return nil
}
