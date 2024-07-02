package guest

import (
	"context"
	"os"
	"path"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

func (g *Guest) copyToGuestRunning(ctx context.Context, dest string, content chan []byte, bot Bot, overrideFolder bool) error {
	if !overrideFolder {
		if isFolder, err := bot.IsFolder(ctx, dest); err != nil {
			return errors.Wrap(err, "")
		} else if isFolder {
			return terrors.ErrFolderExists
		}
	}

	if err := bot.RemoveAll(ctx, dest); err != nil {
		return errors.Wrap(err, "")
	}

	if err := bot.MakeDirectory(ctx, path.Dir(dest), true); err != nil {
		return err
	}

	src, err := bot.OpenFile(ctx, dest, "w")
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer src.Close(ctx)

	for {
		buffer, ok := <-content
		if !ok {
			return nil
		}
		if _, err = src.Write(ctx, buffer); err != nil {
			return errors.Wrap(err, "")
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
			return terrors.ErrFolderExists
		}
	}

	if err := gfx.Remove(dest); err != nil {
		return errors.Wrap(err, "")
	}

	if err := gfx.MakeDirectory(path.Dir(dest), true); err != nil {
		return err
	}

	f, err := os.CreateTemp(os.TempDir(), "toCopy-*")
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer os.Remove(f.Name())
	defer f.Close()

	for {
		buffer, ok := <-content
		if !ok {
			break
		}
		if _, err = f.Write(buffer); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return errors.Wrap(gfx.Upload(f.Name(), dest), "gfx upload error")
}

func (g *Guest) getGfx(dest string) (guestfs.Guestfs, error) {
	vol, err := g.Vols.GetMntVol(dest)
	if err != nil {
		return nil, err
	}
	return vol.GetGfx()
}
