package local

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	"github.com/cockroachdb/errors"
	stotypes "github.com/projecteru2/resource-storage/storage/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	"github.com/projecteru2/yavirt/internal/volume/base"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

var (
	//go:embed templates/disk.xml
	diskXML string
	tmpl    *template.Template
)

// Volume .
// etcd keys:
//
//	/vols/<vol id>
type Volume struct {
	base.Volume            `mapstructure:",squash"`
	stotypes.VolumeBinding `mapstructure:",squash"`

	Format         string       `json:"format" mapstructure:"format"`
	SnapIDs        []string     `json:"snaps" mapstructure:"snaps"`
	BaseSnapshotID string       `json:"base_snapshot_id" mapstructure:"base_snapshot_id"`
	Snaps          Snapshots    `json:"-" mapstructure:"-"`
	flock          *utils.Flock `json:"-" mapstructure:"-"`
}

// LoadVolume loads data from etcd
func LoadVolume(id string) (*Volume, error) {
	var vol = NewVolume()
	vol.ID = id

	if err := meta.Load(vol); err != nil {
		return nil, err
	}

	return vol, vol.LoadSnapshots()
}

// NewVolumeFromStr
// format: `src:dst[:flags][:size][:read_IOPS:write_IOPS:read_bytes:write_bytes]`
// example: `/source:/dir0:rw:1024:1000:1000:10M:10M`
func NewVolumeFromStr(s string) (*Volume, error) {
	vb, err := stotypes.NewVolumeBinding(s)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Format:        VolQcow2Format,
		Volume:        *base.New(),
		VolumeBinding: *vb,
	}, nil
}

// NewSysVolume .
func NewSysVolume(cap int64, imageName string) *Volume {
	vol := NewVolume()
	vol.SysImage = imageName
	vol.Flags = "rws"
	vol.SizeInBytes = cap
	return vol
}

// NewDataVolume .
func NewDataVolume(mnt string, cap int64) (*Volume, error) {
	mnt = strings.TrimSpace(mnt)

	src, dest := utils.PartRight(mnt, ":")
	src = strings.TrimSpace(src)
	dest = filepath.Join("/", strings.TrimSpace(dest))

	if len(src) > 0 {
		src = filepath.Join("/", src)
	}

	var vol = NewVolume()
	vol.Source = src
	vol.Destination = dest
	vol.Flags = "rw"
	vol.SizeInBytes = cap

	return vol, vol.Check()
}

func NewVolume() *Volume {
	return &Volume{
		Volume: *base.New(),
		Format: VolQcow2Format,
	}
}

func (v *Volume) Lock() error {
	fn := fmt.Sprintf("vol_%s.flock", v.ID)
	fpth := filepath.Join(configs.Conf.VirtFlockDir, fn)
	v.flock = utils.NewFlock(fpth)
	if err := v.flock.Trylock(); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (v *Volume) Unlock() {
	v.flock.Close()
}

func (v *Volume) QemuImagePath() string {
	return v.Filepath()
}

func (v *Volume) GetSize() int64 {
	return v.SizeInBytes
}

func (v *Volume) SetSize(sz int64) {
	v.SizeInBytes = sz
}

// Load .
func (v *Volume) LoadSnapshots() (err error) {
	if v.Snaps, err = LoadSnapshots(v.SnapIDs); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (v *Volume) NewSnapshotAPI() base.SnapshotAPI {
	return newSnapshotAPI(v)
}

// AppendSnaps .
func (v *Volume) AppendSnaps(snaps ...*Snapshot) error {
	if v.Snaps.Len()+len(snaps) > configs.Conf.MaxSnapshotsCount {
		return errors.Wrapf(terrors.ErrTooManyVolumes, "at most %d", configs.Conf.MaxSnapshotsCount)
	}

	res := Snapshots(snaps)

	v.Snaps.Append(snaps...)

	v.SnapIDs = append(v.SnapIDs, res.IDs()...)

	return nil
}

// RemoveSnaps Remove snapshots meta by preserving the order.
func (v *Volume) RemoveSnap(snapID string) {
	keep := 0

	for i := 0; i < v.Snaps.Len(); i++ {
		if v.Snaps[i].ID == snapID {
			continue
		}

		v.Snaps[keep] = v.Snaps[i]
		v.SnapIDs[keep] = v.SnapIDs[i]
		keep++
	}

	v.Snaps = v.Snaps[:keep]
	v.SnapIDs = v.SnapIDs[:keep]
}

// Save updates metadata to persistence store.
func (v *Volume) Save() error {
	if v.GetVer() == 0 {
		return meta.Create(meta.Resources{v})
	}
	return meta.Save(meta.Resources{v})
}

func (v *Volume) GetMountDir() string {
	if len(v.Destination) > 0 {
		return v.Destination
	}
	return "/"
}

func (v *Volume) String() string {
	var mnt = "/"
	if len(v.Destination) > 0 {
		mnt = v.Destination
	}
	return fmt.Sprintf("%s, %s, %s:%s, size: %d", v.Filepath(), v.Status, v.GuestID, mnt, v.SizeInBytes)
}

// Filepath .
func (v *Volume) Filepath() string {
	if len(v.Source) > 0 {
		return filepath.Join(v.Source, v.Name())
	}
	return v.JoinVirtPath(v.Name())
}

// Name .
func (v *Volume) Name() string {
	ty := "dat"
	if v.IsSys() {
		ty = "sys"
	}
	return fmt.Sprintf("%s-%s.vol", ty, v.ID)
}

// Check .
func (v *Volume) Check() error {
	switch {
	case v.SizeInBytes < configs.Conf.Resource.MinVolumeCap || v.SizeInBytes > configs.Conf.Resource.MaxVolumeCap:
		return errors.Wrapf(terrors.ErrInvalidValue, "capacity: %d", v.SizeInBytes)
	case v.Source == "/":
		return errors.Wrapf(terrors.ErrInvalidValue, "host dir: %s", v.Source)
	case v.Destination == "/":
		return errors.Wrapf(terrors.ErrInvalidValue, "mount dir: %s", v.Destination)
	default:
		if _, err := os.Stat(v.Filepath()); err == nil {
			return interutils.Check(context.Background(), v.Filepath())
		}
		return nil
	}
}

// Repair .
func (v *Volume) Repair() error {
	return interutils.Repair(context.Background(), v.Filepath())
}

// IsSys .
func (v *Volume) IsSys() bool {
	return strings.Contains(v.Flags, "s")
}

func (v *Volume) PrepareSysDisk(ctx context.Context, img *vmitypes.Image, _ ...base.Option) error {
	if !v.IsSys() {
		panic("not a sys disk")
	}
	rc, err := vmiFact.Pull(ctx, img, vmitypes.PullPolicyAlways)
	if err != nil {
		return errors.Wrapf(err, "failed to pull image %s: %s", img.Fullname(), err)
	}
	interutils.EnsureReaderClosed(rc)
	if err := sh.Copy(img.Filepath(), v.Filepath()); err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (v *Volume) PrepareDataDisk(_ context.Context) error {
	if v.IsSys() {
		panic("not a data disk")
	}
	var path = v.Filepath()
	return interutils.CreateImage(context.TODO(), VolQcow2Format, path, v.SizeInBytes)
}

// Cleanup deletes the qcow2 file
func (v *Volume) Cleanup() error {
	return sh.Remove(v.Filepath())
}

func (v *Volume) CaptureImage(imgName string) (uimg *vmitypes.Image, err error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "local-capture-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	orig := filepath.Join(tmpDir, "vol.img")
	if err := sh.Copy(v.Filepath(), orig); err != nil {
		return nil, errors.Wrap(err, "")
	}
	var gfs guestfs.Guestfs
	if gfs, err = gfsx.New(orig); err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer gfs.Close()
	if err = base.ResetUserImage(gfs); err != nil {
		return nil, errors.Wrap(err, "")
	}

	uimg, err = vmiFact.NewImage(imgName)
	if err != nil {
		return nil, err
	}
	rc, err := vmiFact.Prepare(context.TODO(), orig, uimg)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	defer interutils.EnsureReaderClosed(rc)
	return uimg, nil
}

func (v *Volume) GenerateXML() ([]byte, error) {
	args := map[string]any{
		"path":       v.Filepath(),
		"dev":        v.Device,
		"read_iops":  fmt.Sprintf("%d", v.ReadIOPS),
		"write_iops": fmt.Sprintf("%d", v.WriteIOPS),
		"read_bps":   fmt.Sprintf("%d", v.ReadBPS),
		"write_bps":  fmt.Sprintf("%d", v.WriteBPS),
	}

	if tmpl == nil {
		t, err := template.New("local-vol-tpl").Parse(diskXML)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		tmpl = t
	}

	var wr bytes.Buffer
	if err := tmpl.Execute(&wr, args); err != nil {
		return nil, errors.Wrap(err, "")
	}
	return wr.Bytes(), nil
}

func (v *Volume) GetGfx() (guestfs.Guestfs, error) {
	return gfsx.New(v.Filepath())
}
