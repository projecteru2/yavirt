package base

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/store"
)

type Volume struct {
	*meta.Generic `mapstructure:",squash"`

	SysImage string `json:"sys_image,omitempty" mapstructure:"sys_image"` // for sys volume
	Device   string `json:"device" mapstructure:"device"`                 // vda, vdb, vdc etc.
	Hostname string `json:"host" mapstructure:"host"`
	GuestID  string `json:"guest" mapstructure:"guest"`
}

func New() *Volume {
	return &Volume{
		Generic: meta.NewGeneric(),
	}
}

// GenerateID .
func (v *Volume) GenerateID() {
	v.ID = idgen.Next()
}

func (v *Volume) GetDevice() string {
	return v.Device
}

func (v *Volume) SetDevice(dev string) {
	v.Device = dev
}

func (v *Volume) SetHostname(name string) {
	v.Hostname = name
}

func (v *Volume) GetHostname() string {
	return v.Hostname
}

func (v *Volume) SetGuestID(id string) {
	v.GuestID = id
}

func (v *Volume) GetGuestID() string {
	return v.GuestID
}

// Delete .
func (v *Volume) Delete(force bool) error {
	if err := v.SetStatus(meta.StatusDestroyed, force); err != nil {
		return errors.Wrap(err, "")
	}

	keys := []string{v.MetaKey()}
	vers := map[string]int64{v.MetaKey(): v.GetVer()}

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}
