package image

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// Command .
func Command() *cli.Command {
	return &cli.Command{
		Name: "image",
		Subcommands: []*cli.Command{
			{
				Name:   "add",
				Flags:  addFlags(),
				Action: run.Run(add),
			},
			{
				Name:   "get",
				Flags:  getFlags(),
				Action: run.Run(get),
			},
			{
				Name:   "rm",
				Flags:  rmFlags(),
				Action: run.Run(rm),
			},
			{
				Name:   "list",
				Flags:  listFlags(),
				Action: run.Run(list),
			},
			{
				Name:   "digest",
				Usage:  "",
				Flags:  digestFlags(),
				Action: run.Run(digest),
			},
		},
	}
}

func listFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "user",
			Usage: "the owner of an image",
		},
	}
}

func rmFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "user",
			Usage: "the owner of an image",
		},
	}
}

func getFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "user",
		},
	}
}

func addFlags() []cli.Flag {
	return []cli.Flag{
		&cli.Int64Flag{
			Name:  "size",
			Value: 0,
		},
		&cli.StringFlag{
			Name:  "path",
			Usage: "Set the file path to the image",
			Value: "",
		},
	}
}

func digestFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "remote",
			Usage: "Set for remote digest, otherwise local",
			Value: false,
		},
	}
}

func list(c *cli.Context, runtime run.Runtime) error {
	imgs, err := models.ListImages(c.String("user"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, img := range imgs {
		fmt.Printf("%s\n", img)
	}

	return nil
}

func get(c *cli.Context, runtime run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	img, err := models.LoadImage(name, c.String("user"))
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("image: %s, user: %s, filepath: %s\n", img.GetName(), img.GetUser(), img.Filepath())

	return nil
}

func add(c *cli.Context, runtime run.Runtime) error {
	name := c.Args().First()
	size := c.Int64("size")
	filePath := c.String("path")

	switch {
	case len(name) < 1:
		return errors.New("image name is required")
	case size < 1:
		return errors.New("--size is required")
	}

	img := models.NewSysImage()
	img.Name = name
	img.Size = size

	if err := img.Create(); err != nil {
		return errors.Trace(err)

	}

	fmt.Printf("image %s created\n", img.Name)

	if len(filePath) > 0 {
		// TODO: add image with file to check hash
		// TODO: or download hash from image-hub
		return nil
	}

	return nil
}

func rm(c *cli.Context, runtime run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	user := c.String("user")
	img, err := models.LoadImage(name, user)
	if err != nil {
		return errors.Trace(err)
	}

	if err := img.Delete(); err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("%s has been deleted\n", img)

	return nil
}

func digest(c *cli.Context, runtime run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	if c.Bool("remote") {
		fmt.Print("Remote digest is not supported yet")
		return nil
	}

	img, err := models.LoadSysImage(name)
	if err != nil {
		return errors.Trace(err)
	}

	if len(img.Hash) > 0 {
		fmt.Printf("hash of %s: %s\n", img.Name, img.Hash)
		return nil
	}

	hash, err := img.UpdateHash()
	if err != nil {
		return errors.Trace(err)
	}

	fmt.Printf("hash of %s: %s\n", img.Name, hash)

	return nil
}
