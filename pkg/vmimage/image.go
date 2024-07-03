package vmimage

import (
	"context"
	"io"

	"github.com/projecteru2/yavirt/pkg/vmimage/types"
)

type Manager interface {
	ListLocalImages(ctx context.Context, user string) ([]*types.Image, error)
	LoadImage(ctx context.Context, imgName string) (*types.Image, error) // create image object and pull the image to local

	Prepare(ctx context.Context, fname string, img *types.Image) (io.ReadCloser, error)
	Pull(ctx context.Context, img *types.Image, pullPolicy types.PullPolicy) (io.ReadCloser, error)
	Push(ctx context.Context, img *types.Image, force bool) (io.ReadCloser, error)
	RemoveLocal(ctx context.Context, img *types.Image) error
	CheckHealth(ctx context.Context) error
}
