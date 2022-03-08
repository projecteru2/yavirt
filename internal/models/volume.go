package model

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/store"
	"github.com/projecteru2/yavirt/util"
)

// Volume .
// etcd keys:
//     /vols/<vol id>
type Volume struct {
	*Generic
	Type           string   `json:"type"`
	MountDir       string   `json:"mount,omitempty"`
	HostDir        string   `json:"host_dir,omitempty"`
	Capacity       int64    `json:"capacity"`
	Format         string   `json:"format"`
	HostName       string   `json:"host"`
	GuestID        string   `json:"guest"`
	ImageName      string   `json:"image,omitempty"`
	SnapIDs        []string `json:"snaps"`
	BaseSnapshotID string   `json:"base_snapshot_id"`

	Snaps Snapshots `json:"-"`
}

// LoadVolume .
func LoadVolume(id string) (*Volume, error) {
	var vol = newVolume()
	vol.ID = id

	if err := meta.Load(vol); err != nil {
		return nil, err
	}

	return vol, vol.Load()
}

// NewDataVolume .
func NewDataVolume(mnt string, cap int64) (*Volume, error) {
	mnt = strings.TrimSpace(mnt)

	src, dest := util.PartRight(mnt, ":")
	src = strings.TrimSpace(src)
	dest = filepath.Join("/", strings.TrimSpace(dest))

	if len(src) > 0 {
		src = filepath.Join("/", src)
	}

	var vol = NewVolume(VolDataType, cap)
	vol.HostDir = src
	vol.MountDir = dest

	return vol, vol.Check()
}

// NewSysVolume .
func NewSysVolume(cap int64, imageName string) *Volume {
	vol := NewVolume(VolSysType, cap)
	vol.ImageName = imageName
	return vol
}

// NewVolume .
func NewVolume(vtype string, cap int64) *Volume {
	var vol = newVolume()
	vol.Type = vtype
	vol.Capacity = cap
	return vol
}

func newVolume() *Volume {
	return &Volume{Generic: newGeneric(), Format: VolQcow2Format}
}

// Load .
func (v *Volume) Load() (err error) {
	if v.Snaps, err = LoadSnapshots(v.SnapIDs); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Delete .
func (v *Volume) Delete(force bool) error {
	if err := v.setStatus(StatusDestroyed, force); err != nil {
		return errors.Trace(err)
	}

	keys := []string{v.MetaKey()}
	vers := map[string]int64{v.MetaKey(): v.GetVer()}

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	return store.Delete(ctx, keys, vers)
}

// Amplify .
func (v *Volume) Amplify(cap int64) error {
	v.Capacity = cap
	return v.Save()
}

// AppendSnaps .
func (v *Volume) AppendSnaps(snaps ...*Snapshot) error {
	if v.Snaps.Len()+len(snaps) > config.Conf.MaxSnapshotsCount {
		return errors.Annotatef(errors.ErrTooManyVolumes, "at most %d", config.Conf.MaxSnapshotsCount)
	}

	res := Snapshots(snaps)

	v.Snaps.append(snaps...)

	v.SnapIDs = append(v.SnapIDs, res.ids()...)

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
	return meta.Save(meta.Resources{v})
}

// MetaKey .
func (v *Volume) MetaKey() string {
	return meta.VolumeKey(v.ID)
}

// GenerateID .
func (v *Volume) GenerateID() {
	v.genID()
}

func (v *Volume) genID() {
	v.ID = idgen.Next()
}

// GetDevicePathBySerialNumber .
func (v *Volume) GetDevicePathBySerialNumber(sn int) string {
	return v.GetDevicePathByName(v.GetDeviceName(sn))
}

// GetDevicePathByName .
func (v *Volume) GetDevicePathByName(name string) string {
	return GetDevicePathByName(name)
}

// GetDeviceName .
func (v *Volume) GetDeviceName(sn int) string {
	return GetDeviceName(sn)
}

// GetDevicePathByName .
func GetDevicePathByName(name string) string {
	return filepath.Join("/dev", name)
}

// GetDeviceName .
func GetDeviceName(sn int) string {
	return fmt.Sprintf("vd%s", string(util.LowerLetters[sn]))
}

func (v *Volume) GetMountDir() string {
	if len(v.MountDir) > 0 {
		return v.MountDir
	}
	return "/"
}

func (v *Volume) String() string {
	var mnt = "/"
	if len(v.MountDir) > 0 {
		mnt = v.MountDir
	}
	return fmt.Sprintf("%s, %s, %s:%s, size: %d", v.Filepath(), v.Status, v.GuestID, mnt, v.Capacity)
}

// Filepath .
func (v *Volume) Filepath() string {
	if len(v.HostDir) > 0 {
		return filepath.Join(v.HostDir, v.Name())
	}
	return v.JoinVirtPath(v.Name())
}

// Name .
func (v *Volume) Name() string {
	return fmt.Sprintf("%s-%s.vol", v.Type, v.ID)
}

// Check .
func (v *Volume) Check() error {
	switch {
	case v.Capacity < config.Conf.MinVolumeCap || v.Capacity > config.Conf.MaxVolumeCap:
		return errors.Annotatef(errors.ErrInvalidValue, "capacity: %d", v.Capacity)
	case v.HostDir == "/":
		return errors.Annotatef(errors.ErrInvalidValue, "host dir: %s", v.HostDir)
	case v.MountDir == "/":
		return errors.Annotatef(errors.ErrInvalidValue, "mount dir: %s", v.MountDir)
	default:
		return nil
	}
}

// IsSys .
func (v *Volume) IsSys() bool {
	return v.Type == VolSysType
}

// LoadVolumes .
func LoadVolumes(ids []string) (vols Volumes, err error) {
	vols = make(Volumes, len(ids))

	for i, id := range ids {
		if vols[i], err = LoadVolume(id); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return vols, nil
}

// Volumes .
type Volumes []*Volume

// Check .
func (vols Volumes) Check() error {
	for _, v := range vols {
		if v == nil {
			return errors.Annotatef(errors.ErrInvalidValue, "nil *Volume")
		}
		if err := v.Check(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// Find .
func (vols Volumes) Find(volID string) (*Volume, error) {
	for _, v := range vols {
		if v.ID == volID {
			return v, nil
		}
	}

	return nil, errors.Annotatef(errors.ErrInvalidValue, "volID %s not exists", volID)
}

func (vols Volumes) resources() meta.Resources {
	var r = make(meta.Resources, len(vols))
	for i, v := range vols {
		r[i] = v
	}
	return r
}

func (vols *Volumes) append(vol ...*Volume) {
	*vols = append(*vols, vol...)
}

func (vols Volumes) setGuestID(id string) {
	for _, vol := range vols {
		vol.GuestID = id
	}
}

func (vols Volumes) setHostName(name string) {
	for _, vol := range vols {
		vol.HostName = name
	}
}

func (vols Volumes) ids() []string {
	var v = make([]string, len(vols))
	for i, vol := range vols {
		v[i] = vol.ID
	}
	return v
}

func (vols Volumes) genID() {
	for _, vol := range vols {
		vol.genID()
	}
}

func (vols Volumes) setStatus(st string, force bool) error {
	for _, vol := range vols {
		if err := vol.setStatus(st, force); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (vols Volumes) deleteKeys() []string {
	var keys = make([]string, len(vols))
	for i, vol := range vols {
		keys[i] = vol.MetaKey()
	}
	return keys
}

// Exists checks the volume if exists, in which mounted the directory.
func (vols Volumes) Exists(mnt string) bool {
	for _, vol := range vols {
		switch {
		case vol.IsSys():
			continue
		case vol.MountDir == mnt:
			return true
		}
	}
	return false
}

// Len .
func (vols Volumes) Len() int {
	return len(vols)
}

// GetMntVol return the vol of a path if exists .
func (vols Volumes) GetMntVol(path string) (*Volume, error) {
	path = filepath.Dir(path)
	if path[0] != '/' {
		return nil, errors.ErrDestinationInvalid
	}

	var sys, maxVol *Volume
	maxLen := -1
	for _, vol := range vols {
		if vol.IsSys() {
			sys = vol
			continue
		}

		mntDirLen := len(vol.MountDir)
		if mntDirLen > maxLen && strings.Index(path, vol.MountDir) == 0 {
			maxLen = mntDirLen
			maxVol = vol
		}
	}

	if maxLen < 1 {
		return sys, nil
	}
	return maxVol, nil
}
