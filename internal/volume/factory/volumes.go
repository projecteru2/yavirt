package factory

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/volume"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/internal/volume/mocks"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

type Volumes []volume.Volume

// Check .
func (vols Volumes) Check() error {
	for _, v := range vols {
		if v == nil {
			return errors.Wrapf(terrors.ErrInvalidValue, "nil *Volume")
		}
		if err := v.Check(); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

// Find .
func (vols Volumes) Find(volID string) (volume.Volume, error) {
	for _, v := range vols {
		if v.GetID() == volID {
			return v, nil
		}
	}

	return nil, errors.Wrapf(terrors.ErrInvalidValue, "volID %s not exists", volID)
}

// Exists checks the volume if exists, in which mounted the directory.
func (vols Volumes) Exists(mnt string) bool {
	for _, vol := range vols {
		switch {
		case vol.IsSys():
			continue
		case vol.GetMountDir() == mnt:
			return true
		}
	}
	return false
}

// Len .
func (vols Volumes) Len() int {
	return len(vols)
}

func (vols Volumes) Resources() meta.Resources {
	var r = make(meta.Resources, 0, len(vols))
	for _, v := range vols {
		// for UT, because can't marshal mock object
		if _, ok := v.(*mocks.Volume); ok {
			continue
		}
		r = append(r, v)
	}
	return r
}

func (vols Volumes) SetDevice() {
	for idx, vol := range vols {
		dev := base.GetDeviceName(idx)
		vol.SetDevice(dev)
	}
}

func (vols Volumes) SetGuestID(id string) {
	for _, vol := range vols {
		vol.SetGuestID(id)
	}
}

func (vols Volumes) SetHostName(name string) {
	for _, vol := range vols {
		vol.SetHostname(name)
	}
}

func (vols Volumes) IDs() []string {
	var v = make([]string, len(vols))
	for i, vol := range vols {
		v[i] = vol.GetID()
	}
	return v
}

func (vols Volumes) GenID() {
	for _, vol := range vols {
		vol.GenerateID()
	}
}

func (vols Volumes) SetStatus(st string, force bool) error {
	for _, vol := range vols {
		if err := vol.SetStatus(st, force); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (vols Volumes) MetaKeys() []string {
	var keys = make([]string, len(vols))
	for i, vol := range vols {
		keys[i] = vol.MetaKey()
	}
	return keys
}

// GetMntVol return the vol of a path if exists .
func (vols Volumes) GetMntVol(path string) (volume.Volume, error) {
	path = filepath.Dir(path)
	if path[0] != '/' {
		return nil, terrors.ErrDestinationInvalid
	}

	// append a `/` to the end.
	// this can avoid the wrong result caused by following example:
	// path: /dst10, mount dir: /dst1
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	var sys, maxVol volume.Volume
	maxLen := -1
	for _, vol := range vols {
		if vol.IsSys() {
			sys = vol
			continue
		}

		mntDirLen := len(vol.GetMountDir())
		mntDir := vol.GetMountDir()
		if !strings.HasSuffix(mntDir, "/") {
			mntDir += "/"
		}
		if mntDirLen > maxLen && strings.Index(path, mntDir) == 0 {
			maxLen = mntDirLen
			maxVol = vol
		}
	}

	if maxLen < 1 {
		return sys, nil
	}
	return maxVol, nil
}
