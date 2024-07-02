package boar

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	vmiFact "github.com/yuyang0/vmimage/factory"
	vmitypes "github.com/yuyang0/vmimage/types"
)

func (svc *Boar) PushImage(ctx context.Context, imgName string, force bool) (rc io.ReadCloser, err error) {
	svc.imageMutex.Lock()
	defer svc.imageMutex.Unlock()

	img, err := vmiFact.NewImage(imgName)
	if err != nil {
		return nil, err
	}
	if rc, err = vmiFact.Push(ctx, img, force); err != nil {
		err = errors.Wrap(err, "")
		return
	}
	return
}

func (svc *Boar) RemoveImage(ctx context.Context, imageName string, force, prune bool) (removed []string, err error) { //nolint
	defer logErr(err)

	img, err := vmiFact.LoadImage(ctx, imageName)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	svc.imageMutex.Lock()
	defer svc.imageMutex.Unlock()

	if err = vmiFact.RemoveLocal(ctx, img); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return []string{img.Fullname()}, nil
}

func (svc *Boar) ListImage(ctx context.Context, filter string) (ans []*vmitypes.Image, err error) {
	defer logErr(err)

	imgs, err := vmiFact.ListLocalImages(ctx, "")
	if err != nil {
		return nil, err
	}

	images := []*vmitypes.Image{}
	if len(filter) < 1 {
		images = imgs
	} else {
		var regExp *regexp.Regexp
		filter = strings.ReplaceAll(filter, "*", ".*")
		if regExp, err = regexp.Compile(fmt.Sprintf("%s%s%s", "^", filter, "$")); err != nil {
			return nil, err
		}

		for _, img := range imgs {
			if regExp.MatchString(img.Fullname()) {
				images = append(images, img)
			}
		}
	}

	return images, err
}

func (svc *Boar) PullImage(ctx context.Context, imgName string) (img *vmitypes.Image, rc io.ReadCloser, err error) {
	svc.imageMutex.Lock()
	defer svc.imageMutex.Unlock()

	img, err = vmiFact.NewImage(imgName)
	if err != nil {
		err = errors.Wrap(err, "")
		return
	}
	if rc, err = vmiFact.Pull(ctx, img, vmitypes.PullPolicyAlways); err != nil {
		err = errors.Wrap(err, "")
		return
	}
	return
}

func (svc *Boar) DigestImage(ctx context.Context, imageName string, local bool) (digest []string, err error) {
	defer logErr(err)

	if !local {
		// TODO: wait for image-hub implementation and calico update
		return []string{""}, nil
	}

	// If not exists return error
	// If exists return digests

	img, err := vmiFact.LoadImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	return []string{img.GetDigest()}, nil
}
