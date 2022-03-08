package guest

import (
	"context"
	"io/ioutil"
	"os"
	"path"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/virt/guestfs"
	"github.com/projecteru2/yavirt/virt/guestfs/gfsx"
)

func (g *Guest) copyToGuestRunning(ctx context.Context, dest string, content chan []byte, bot Bot, overrideFolder bool) error {
	if !overrideFolder {
		if isFolder, err := bot.IsFolder(ctx, dest); err != nil {
			return errors.Trace(err)
		} else if isFolder {
			return errors.ErrFolderExists
		}
	}

	if err := bot.RemoveAll(ctx, dest); err != nil {
		return errors.Trace(err)
	}

	if err := bot.MakeDirectory(ctx, path.Dir(dest), true); err != nil {
		return err
	}

	src, err := bot.OpenFile(dest, "w")
	if err != nil {
		return errors.Trace(err)
	}
	defer src.Close()

	for {
		buffer, ok := <-content
		if !ok {
			return nil
		}
		if _, err = src.Write(buffer); err != nil {
			return errors.Trace(err)
		}
	}
}

func (g *Guest) copyToGuestNotRunning(dest string, content chan []byte, overrideFolder bool, gfx guestfs.Guestfs) error {
	if !overrideFolder {
		isDir, err := gfx.IsDir(dest)
		if err != nil {
			return err
		}
		if isDir {
			return errors.ErrFolderExists
		}
	}

	if err := gfx.Remove(dest); err != nil {
		return errors.Trace(err)
	}

	if err := gfx.MakeDirectory(path.Dir(dest), true); err != nil {
		return err
	}

	f, err := ioutil.TempFile(os.TempDir(), "toCopy-*")
	if err != nil {
		return errors.Trace(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	for {
		buffer, ok := <-content
		if !ok {
			break
		}
		if _, err = f.Write(buffer); err != nil {
			return errors.Trace(err)
		}
	}

	return errors.Trace(gfx.Upload(f.Name(), dest))
}

func (g *Guest) getGfx(dest string) (guestfs.Guestfs, error) {
	vol, err := g.Vols.GetMntVol(dest)
	if err != nil {
		return nil, err
	}

	return gfsx.New(vol.Filepath())
}
