package rbd

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	_ "embed"

	"text/template"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/gfsx"
	"github.com/projecteru2/yavirt/internal/volume/base"
	vmiFact "github.com/projecteru2/yavirt/pkg/vmimage/factory"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
	libguestfs "github.com/projecteru2/yavirt/third_party/guestfs"
	rbdtypes "github.com/yuyang0/resource-rbd/rbd/types"
)

var (
	//go:embed templates/disk.xml
	diskXML string
	tmpl    *template.Template
)

type Volume struct {
	base.Volume            `mapstructure:",squash"`
	rbdtypes.VolumeBinding `mapstructure:",squash"`
}

func New() *Volume {
	return &Volume{
		Volume: *base.New(),
	}
}

func NewFromStr(ss string) (*Volume, error) {
	vb, err := rbdtypes.NewVolumeBinding(ss)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Volume:        *base.New(),
		VolumeBinding: *vb,
	}, nil
}

func (v *Volume) Name() string {
	return fmt.Sprintf("rbd-%s", v.ID)
}

func (v *Volume) QemuImagePath() string {
	rbdDisk := fmt.Sprintf("rbd:%s/%s:id=%s", v.Pool, v.Image, configs.Conf.Storage.Ceph.Username)
	return rbdDisk
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

func (v *Volume) IsSys() bool {
	return strings.Contains(v.Flags, "s")
}

func (v *Volume) PrepareSysDisk(ctx context.Context, img *vmitypes.Image, opts ...base.Option) error {
	if !v.IsSys() {
		panic("not a sys disk")
	}
	logger := log.WithFunc("Prepare")
	optVal := &base.OptionValue{}
	for _, opt := range opts {
		opt(optVal)
	}
	// check if this rbd already exists
	client, err := GetRBDConn()
	if err != nil {
		return err
	}
	defer client.Shutdown()

	ioctx, err := client.OpenIOContext(v.Pool)
	if err != nil {
		return err
	}
	defer ioctx.Destroy()
	// check if RBD already exist
	exist, err := v.exist(ioctx)
	if err != nil {
		return err
	}
	if exist {
		logger.Infof(ctx, "RBD image %s already exist, skip creating", v.Image)
		return nil
	}

	rbdDisk := fmt.Sprintf("rbd:%s/%s:id=%s", v.Pool, v.Image, configs.Conf.Storage.Ceph.Username)
	// try to create rbd from image snapshot
	if err := v.createSysRBDFromSnap(ctx, client, ioctx, img); err != nil {
		logger.Warnf(ctx, "failed to create rbd(%s) from image(%s) snaoshot: %s", rbdDisk, img.Fullname(), err)

		// try to create rbd from image file
		if err := v.createSysRBDFromImageFile(ctx, img); err != nil {
			return err
		}
	}
	if err := interutils.ResizeImage(ctx, rbdDisk, v.SizeInBytes); err != nil {
		if err := rbd.RemoveImage(ioctx, v.Image); err != nil {
			logger.Warnf(ctx, "[rollback] failed to delete rbd %s: %s", rbdDisk, err)
		}
		return errors.Wrapf(err, "failed to resize rbd")
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

func (v *Volume) GenerateXML() ([]byte, error) {
	// prepare monitor addresses
	cephMonitorAddrs := []map[string]string{}
	for _, addr := range configs.Conf.Storage.Ceph.MonitorAddrs {
		parts := strings.Split(addr, ":")
		d := map[string]string{
			"host": parts[0],
			"port": parts[1],
		}
		cephMonitorAddrs = append(cephMonitorAddrs, d)
	}

	args := map[string]any{
		"source":       v.GetSource(),
		"dev":          v.Device,
		"monitorAddrs": cephMonitorAddrs,
		"username":     configs.Conf.Storage.Ceph.Username,
		"secretUUID":   configs.Conf.Storage.Ceph.SecretUUID,
	}
	if tmpl == nil {
		t, err := template.New("rbd-vol-tpl").Parse(diskXML)
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

func GetRBDConn() (*rados.Conn, error) {
	// todo... 需要安装配置 rbd rados连接
	conn, err := rados.NewConnWithUser(configs.Conf.Storage.Ceph.Username)
	if err != nil {
		return nil, err
	}
	if err := conn.ReadDefaultConfigFile(); err != nil {
		return nil, err
	}
	if err := conn.Connect(); err != nil {
		return nil, err
	}
	// defer conn.Shutdown()
	return conn, nil
}

// Cleanup is mainly used to clean old sys disk before reiniitializing sys disk
func (v *Volume) Cleanup() error {
	return nil
}

func (v *Volume) exist(ioctx *rados.IOContext) (bool, error) {
	img, err := rbd.OpenImageReadOnly(ioctx, v.Image, rbd.NoSnapshot)
	if err == nil {
		img.Close()
		return true, nil
	}
	if strings.Contains(err.Error(), "image not found") {
		return false, nil
	}
	return false, err
}

func (v *Volume) CaptureImage(imgName string) (uimg *vmitypes.Image, err error) { //nolint
	// rbdDisk := fmt.Sprintf("rbd:%s/%s:id=%s", v.Pool, v.Image, configs.Conf.Storage.Ceph.Username)

	// log.Warnf("[Prepare] failed to clone snapshot(image %s, rbd: %s): %v", img.RBDName(), rbdDisk, err)

	// if err := util.DumpBLK(context.TODO(), img.Filepath(), rbdDisk); err != nil {
	// 	return errors.Wrap(err, "")
	// }
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
	opts := &libguestfs.OptargsAdd_drive{
		Readonly_is_set: false,
		Format_is_set:   true,
		Format:          "raw",
		Protocol_is_set: true,
		Protocol:        "rbd",
		Server_is_set:   true,
		// TODO add servers, user and secret for ceph
		Server:          []string{},
		Username_is_set: true,
		// User: "eru",
		Secret_is_set: true,
		Secret:        "",
	}
	return gfsx.NewFromOpts(v.GetSource(), opts)
}

func (v *Volume) createSysRBDFromSnap(
	_ context.Context, client *rados.Conn,
	ioctx *rados.IOContext, img *vmitypes.Image,
) (err error) {
	snapshot := img.Snapshot
	var srcPool, srcImgName, snapName string

	if img.Snapshot != "" {
		srcPool, srcImgName, snapName, err = parseSnapName(snapshot)
	} else {
		// this is for compatibility
		srcPool = configs.Conf.Storage.Ceph.Username
		srcImgName = img.RBDName()
		snapName = "latest"
	}
	if err != nil {
		return errors.Wrapf(err, "failed to parse snapshot name %s or image %s", snapshot, img.Fullname())
	}

	srcIOCtx := ioctx
	if srcPool != v.Pool {
		// try to create rbd from image rbd snapshot
		if srcIOCtx, err = client.OpenIOContext(srcPool); err != nil {
			return err
		}
		defer srcIOCtx.Destroy()
	}
	if err = rbd.CloneImage(srcIOCtx, srcImgName, snapName, ioctx, v.Image, rbd.NewRbdImageOptions()); err == nil {
		return errors.Wrapf(err, "failed to clone image")
	}
	return nil
}

func (v *Volume) createSysRBDFromImageFile(ctx context.Context, img *vmitypes.Image) error {
	rbdDisk := fmt.Sprintf("rbd:%s/%s:id=%s", v.Pool, v.Image, configs.Conf.Storage.Ceph.Username)
	// try to create rbd from image file
	rc, err := vmiFact.Pull(ctx, img, vmitypes.PullPolicyAlways)
	if err != nil {
		return errors.Wrapf(err, "failed to pull image %s", img.Fullname())
	}
	interutils.EnsureReaderClosed(rc)
	if err := interutils.WriteBLK(ctx, img.Filepath(), rbdDisk, true); err != nil {
		return errors.Wrap(err, "failed to write image file to rbd")
	}
	return nil
}
