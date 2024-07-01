package factory

import (
	"context"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/meta"
	gfsmocks "github.com/projecteru2/yavirt/internal/virt/guestfs/mocks"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/internal/volume/local"
	"github.com/projecteru2/yavirt/internal/volume/rbd"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func LoadVolumes(ids []string) (vols Volumes, err error) {
	logger := log.WithFunc("volume.factory.LoadVolumes")
	vols = make(Volumes, len(ids))

	for i, id := range ids {
		key := meta.VolumeKey(id)
		rawVal, ver, err := meta.LoadRaw(key)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		var vol volume.Volume
		if _, ok := rawVal["pool"]; ok {
			vol = rbd.New()
		} else {
			vol = local.NewVolume()
		}

		if err := mapstructure.Decode(rawVal, &vol); err != nil {
			return vols, err
		}
		vol.SetVer(ver)
		// FIXME: just for compatibility, when all existing volumes contain device filed,
		// we can delete the following code
		if vol.GetDevice() == "" {
			logger.Warnf(context.TODO(), "[BUG] volume(%s) has no device", vol.GetID())
			vol.SetDevice(base.GetDeviceName(i))
			if err := vol.Save(); err != nil {
				logger.Errorf(context.TODO(), err, "Failed to save volume(%v)", vol)
			}
		}
		vols[i] = vol
	}
	return vols, nil
}

// NewMockedVolume for unit test.
func NewMockedVolume() (volume.Volume, *gfsmocks.Guestfs) {
	gfs := &gfsmocks.Guestfs{}
	vol := local.NewSysVolume(utils.GB, "unitest-image")
	return vol, gfs
}
