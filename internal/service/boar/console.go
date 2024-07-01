package boar

import (
	"context"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	intertypes "github.com/projecteru2/yavirt/internal/types"
)

// AttachGuest .
func (svc *Boar) AttachGuest(ctx context.Context, id string, stream io.ReadWriteCloser, flags intertypes.OpenConsoleFlags) (err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if g.LambdaOption != nil {
		if err = g.Wait(meta.StatusRunning, false); err != nil {
			return errors.Wrap(err, "")
		}
		flags.Commands = g.LambdaOption.Cmd
	}

	return g.AttachConsole(ctx, stream, flags)
}
