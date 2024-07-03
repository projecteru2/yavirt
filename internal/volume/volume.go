package volume

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/volume/base"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

type Volume interface { //nolint:interfacebloat
	meta.GenericInterface

	// getters
	Name() string
	QemuImagePath() string
	GetMountDir() string
	GetSize() int64
	GetDevice() string
	GetHostname() string
	GetGuestID() string
	// setters
	SetDevice(dev string)
	SetHostname(name string)
	SetGuestID(id string)
	SetSize(size int64)
	GenerateID()

	// Note: caller should call dom.Free() to release resource
	Check() error
	Repair() error
	IsSys() bool
	// prepare the volume, run before create guest.
	PrepareSysDisk(context.Context, *vmitypes.Image, ...base.Option) error
	PrepareDataDisk(context.Context) error

	GenerateXML() ([]byte, error)
	Cleanup() error
	// delete data in store
	Delete(force bool) error
	CaptureImage(imgName string) (*vmitypes.Image, error)
	// Save data to store
	Save() error

	Lock() error
	Unlock()

	GetGfx() (guestfs.Guestfs, error)

	NewSnapshotAPI() base.SnapshotAPI
}

func WithLocker(vol Volume, fn func() error) error {
	if err := vol.Lock(); err != nil {
		return errors.Wrap(err, "")
	}
	defer vol.Unlock()
	return fn()
}
