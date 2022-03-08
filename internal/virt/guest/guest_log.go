package guest

import (
	"context"
	"io"

	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func (g *Guest) logRunning(ctx context.Context, bot Bot, n int, logPath string, dest io.WriteCloser) error {
	src, err := bot.OpenFile(logPath, "r")
	if err != nil {
		return errors.Trace(err)
	}
	defer src.Close()

	if n < 0 { // read all
		_, err = util.CopyIO(ctx, dest, src)
		return err
	}

	content, err := src.Tail(n)
	if err != nil {
		return err
	}
	_, err = dest.Write(content)

	return err
}

func (g *Guest) logStopped(n int, logPath string, dest io.WriteCloser, gfx guestfs.Guestfs) error {
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
