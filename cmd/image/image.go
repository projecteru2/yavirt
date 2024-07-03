package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"github.com/urfave/cli/v2"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/cmd/run"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/utils"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
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
				Action: run.Run(get),
			},
			{
				Name:   "rm",
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
			{
				Name:   "rbd",
				Usage:  "",
				Flags:  rbdFlags(),
				Action: run.Run(rbdAction),
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

func rbdFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "update",
			Usage: "update rbd for image",
			Value: false,
		},
	}
}

func list(c *cli.Context, _ run.Runtime) error {
	imgs, err := vmiFact.ListLocalImages(c.Context, c.String("user"))
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, img := range imgs {
		fmt.Printf("%s\n", img.Fullname())
	}

	return nil
}

func get(c *cli.Context, _ run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}
	img, err := vmiFact.LoadImage(c.Context, name)
	if err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("image: %s, filepath: %s\n", img.Fullname(), img.Filepath())

	return nil
}

func add(c *cli.Context, _ run.Runtime) error {
	name := c.Args().First()
	size := c.Int64("size")
	filePath := c.String("path")

	switch {
	case len(name) < 1:
		return errors.New("image name is required")
	case size < 1:
		return errors.New("--size is required")
	}
	img, err := vmiFact.NewImage(name)
	if err != nil {
		return err
	}
	fmt.Printf("*** Prepare image\n")
	if rc, err := vmiFact.Prepare(c.Context, filePath, img); err != nil {
		return errors.Wrap(err, "")
	} else { //nolint
		defer rc.Close()
		if _, err := io.Copy(os.Stdout, rc); err != nil {
			return errors.Wrap(err, "")
		}
	}

	fmt.Printf("*** Push image\n")
	if rc, err := vmiFact.Push(c.Context, img, false); err != nil {
		return errors.Wrap(err, "")
	} else { //nolint
		defer rc.Close()
		if _, err = io.Copy(os.Stdout, rc); err != nil {
			return errors.Wrap(err, "")
		}
	}

	fmt.Printf("image %s created\n", img.Fullname())
	return nil
}

func rm(c *cli.Context, _ run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	img, err := vmiFact.LoadImage(c.Context, name)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := vmiFact.RemoveLocal(c.Context, img); err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("%s has been deleted\n", img.Fullname())

	return nil
}

func digest(c *cli.Context, _ run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	if c.Bool("remote") {
		fmt.Print("Remote digest is not supported yet")
		return nil
	}

	img, err := vmiFact.LoadImage(c.Context, name)
	if err != nil {
		return errors.Wrap(err, "")
	}

	fmt.Printf("hash of %s: %s\n", img.Fullname(), img.GetDigest())

	return nil
}

func createAndProtectSnapshot(pool, imgName, snapName string, update bool) error {
	conn, err := rados.NewConnWithUser(configs.Conf.Storage.Ceph.Username)
	if err != nil {
		return err
	}
	if err := conn.ReadDefaultConfigFile(); err != nil {
		return err
	}
	if err := conn.Connect(); err != nil {
		return err
	}
	defer conn.Shutdown()
	ctx, err := conn.OpenIOContext(pool)
	if err != nil {
		return err
	}
	defer ctx.Destroy()

	rbdImage, err := rbd.OpenImage(ctx, imgName, rbd.NoSnapshot)
	if err != nil {
		return err
	}
	if update {
		// rename snapshot
		oldSnap := rbdImage.GetSnapshot(snapName)
		oldName := fmt.Sprintf("%s_%d", snapName, time.Now().UnixNano())
		if err := oldSnap.Rename(oldName); err != nil {
			return err
		}
	}
	snapshot, err := rbdImage.CreateSnapshot(snapName)
	if err != nil {
		return err
	}
	return snapshot.Protect()
}

func rbdAction(c *cli.Context, _ run.Runtime) error {
	name := c.Args().First()
	if len(name) < 1 {
		return errors.New("image name is required")
	}

	img, err := vmiFact.LoadImage(c.Context, name)
	if err != nil {
		return errors.Wrap(err, "")
	}

	rbdDisk := fmt.Sprintf("rbd:eru/%s:id=%s", img.RBDName(), configs.Conf.Storage.Ceph.Username)
	if c.Bool("update") {
		if err := utils.ForceWriteBLK(context.TODO(), img.Filepath(), rbdDisk); err != nil {
			return errors.Wrap(err, "")
		}
	} else {
		if err := utils.WriteBLK(context.TODO(), img.Filepath(), rbdDisk, true); err != nil {
			return errors.Wrap(err, "")
		}
	}
	if err = createAndProtectSnapshot("eru", img.RBDName(), "latest", c.Bool("update")); err != nil {
		return errors.Wrap(err, "")
	}
	fmt.Printf("write %s to %s successfully", name, rbdDisk)

	return nil
}
