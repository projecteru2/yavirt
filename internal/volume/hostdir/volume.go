package hostdir

import (
	"context"
	"fmt"
	"path/filepath"

	_ "embed"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/utils/template"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
	hostdirTypes "github.com/yuyang0/resource-hostdir/hostdir/types"
)

var (
	//go:embed templates/disk.xml
	diskXML string
)

type Volume struct {
	base.Volume                `mapstructure:",squash"`
	hostdirTypes.VolumeBinding `mapstructure:",squash"`
}

func New() *Volume {
	return &Volume{
		Volume: *base.New(base.VolumeTypeHostDir),
	}
}

func NewFromStr(ss string) (*Volume, error) {
	vb, err := hostdirTypes.NewVolumeBinding(ss)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Volume:        *base.New(base.VolumeTypeHostDir),
		VolumeBinding: *vb,
	}, nil
}

func (v *Volume) Name() string {
	return fmt.Sprintf("hostdir-%s", v.ID)
}

func (v *Volume) QemuImagePath() string {
	return ""
}

func (v *Volume) GetSize() int64 {
	return v.SizeInBytes
}

func (v *Volume) SetSize(sz int64) {
	v.SizeInBytes = sz
}

func (v *Volume) GetMountDir() string {
	return v.Destination
}

func (v *Volume) GetXMLQStr() string {
	return fmt.Sprintf("//devices/filesystem[target[@dir='%s']]", v.Destination)
}

func (v *Volume) IsSys() bool {
	return false
}

func (v *Volume) PrepareSysDisk(_ context.Context, _ *vmitypes.Image, _ ...base.Option) error {
	if !v.IsSys() {
		panic("not a sys disk")
	}
	return nil
}

func (v *Volume) PrepareDataDisk(_ context.Context) error {
	if v.IsSys() {
		panic("not a data disk")
	}
	return nil
}

// qemu-img doesn't support checking rbd
func (v *Volume) Check() error {
	return nil
}

func (v *Volume) Repair() error {
	return nil
}

// mount -t virtiofs mount_tag /mnt/mount/path
// we use destination as mount tag
func (v *Volume) Mount(ctx context.Context, ga agent.Interface, _ string) error {
	const (
		fs         = "virtiofs"
		backupDump = 0
		fsckPass   = 0
	)
	devPath := v.Destination
	mountDir := v.GetMountDir()
	if err := base.Mount(ctx, ga, devPath, mountDir, fs); err != nil {
		return errors.Wrap(err, "")
	}
	if err := base.SaveFstab(ctx, ga, devPath, mountDir, fs, backupDump, fsckPass); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (v *Volume) Umount(ctx context.Context, ga agent.Interface) error {
	return base.UmountDevice(ctx, ga, v.Destination)
}

func (v *Volume) AmplifyOffline(_ context.Context, _ int64) error {
	return nil
}

func (v *Volume) AmplifyOnline(_ int64, _ libvirt.Domain, _ agent.Interface) error {
	return nil
}

func (v *Volume) GenerateXML() ([]byte, error) {
	args := map[string]any{
		"src": v.Source,
		"dst": v.Destination,
	}
	tmplFile := filepath.Join(configs.Conf.VirtTmplDir, "hostdir.xml")
	return template.Render(tmplFile, diskXML, args)
}

// Cleanup is mainly used to clean old sys disk before reiniitializing sys disk
func (v *Volume) Cleanup() error {
	return nil
}

func (v *Volume) CaptureImage(imgName string) (uimg *vmitypes.Image, err error) { //nolint
	return
}

func (v *Volume) Save() error {
	if v.GetVer() == 0 {
		return meta.Create(meta.Resources{v})
	}
	return meta.Save(meta.Resources{v})
}

func (v *Volume) Lock() error {
	return nil
}

func (v *Volume) Unlock() {}

func (v *Volume) NewSnapshotAPI() base.SnapshotAPI {
	return newSnapshotAPI(v)
}

func (v *Volume) GetGfx() (guestfs.Guestfs, error) {
	// opts := &libguestfs.OptargsAdd_drive{
	// 	Readonly_is_set: false,
	// 	Format_is_set:   true,
	// 	Format:          "raw",
	// 	Protocol_is_set: true,
	// 	Protocol:        "rbd",
	// 	Server_is_set:   true,
	// 	// TODO add servers, user and secret for ceph
	// 	Server:          []string{},
	// 	Username_is_set: true,
	// 	// User: "eru",
	// 	Secret_is_set: true,
	// 	Secret:        "",
	// }
	// return gfsx.NewFromOpts(v.GetSource(), opts)
	return nil, nil //nolint:nilnil
}
