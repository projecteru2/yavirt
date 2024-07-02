package guest

import (
	"context"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
)

func (g *Guest) logRunning(ctx context.Context, bot Bot, n int, logPath string, dest io.Writer) error {
	src, err := bot.OpenFile(ctx, logPath, "r")
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer src.Close(ctx)

	if n < 0 { // read all
		_, err = src.CopyTo(ctx, dest)
		return err
	}

	content, err := src.Tail(ctx, n)
	if err != nil {
		return err
	}
	_, err = dest.Write(content)

	return err
}

func (g *Guest) logStopped(n int, logPath string, dest io.Writer, gfx guestfs.Guestfs) error {
	if n < 0 { // Read all
		content, err := gfx.Read(logPath)
		if err != nil {
			return err
		}
		_, err = dest.Write(content)
		return err
	}

	logs, err := gfx.Tail(n, logPath)
	if err != nil {
		return err
	}
	for _, s := range logs {
		if _, err := dest.Write(append([]byte(s), byte('\n'))); err != nil {
			return err
		}
	}

	return nil
}
