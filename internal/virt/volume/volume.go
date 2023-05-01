package volume

import (
	"time"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/virt/agent"
	"github.com/projecteru2/yavirt/internal/virt/domain"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/libvirt"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// Virt .
type Virt interface { //nolint
	Mount(ga agent.Interface, devPath string) error
	IsSys() bool
	Amplify(cap int64, dom domain.Domain, ga agent.Interface, devPath string) (delta int64, err error)
	Filepath() string
	Model() *models.Volume
	Alloc(models.Image) error
	Undefine() error
	ConvertImage(user, name string) (uimg *models.UserImage, err error)
	Attach(dom domain.Domain, ga agent.Interface, devName string) (rollback func(), err error)
	Check() error
	Repair() error
	CreateSnapshot() error
	CommitSnapshot(snapID string) error
	CommitSnapshotByDay(day int) error
	RestoreSnapshot(snapID string) error
}

// Volume .
type Volume struct {
	*models.Volume
	newBot func(*Volume) (Bot, error)
}

// New .
func New(mod *models.Volume) *Volume {
	return &Volume{
		Volume: mod,
		newBot: newVirtVol,
	}
}

// Model .
func (vol *Volume) Model() *models.Volume {
	return vol.Volume
}

// Undefine .
func (vol *Volume) Undefine() error {
	if err := vol.botOperate(func(bot Bot) error {
		return bot.DeleteAllSnapshots()
	}); err != nil {
		return errors.Trace(err)
	}

	if err := vol.Model().Save(); err != nil {
		return errors.Trace(err)
	}

	return vol.botOperate(func(bot Bot) error {
		return bot.Undefine()
	})
}

// ConvertImage .
func (vol *Volume) ConvertImage(user, name string) (uimg *models.UserImage, err error) {
	if !vol.IsSys() {
		return nil, errors.Annotatef(errors.ErrNotSysVolume, vol.ID)
	}

	vol.botOperate(func(bot Bot) error { //nolint
		uimg, err = bot.ConvertUserImage(user, name)
		return err
	})

	return
}

// Attach .
func (vol *Volume) Attach(dom domain.Domain, ga agent.Interface, devName string) (rollback func(), err error) {
	if rollback, err = vol.create(); err != nil {
		return
	}

	defer func() {
		if err != nil {
			rollback()
		}
		rollback = nil
	}()

	var st libvirt.DomainState
	if st, err = dom.AttachVolume(vol.Filepath(), devName); err == nil && st == libvirt.DomainRunning {
		err = vol.Mount(ga, models.GetDevicePathByName(devName))
	}

	return
}

func (vol *Volume) create() (func(), error) {
	if err := vol.Alloc(nil); err != nil {
		return nil, errors.Trace(err)
	}

	if err := vol.Volume.Save(); err != nil {
		if ue := vol.Undefine(); ue != nil {
			err = errors.Wrap(err, ue)
		}
		return nil, errors.Trace(err)
	}

	rb := func() {
		if err := vol.Undefine(); err != nil {
			log.ErrorStack(err)
			return
		}
		if err := vol.Delete(true); err != nil {
			log.ErrorStack(err)
		}
	}

	return rb, nil
}

// Alloc .
func (vol *Volume) Alloc(img models.Image) error {
	return vol.botOperate(func(bot Bot) error {
		switch vol.Type {
		case models.VolSysType:
			if img == nil {
				return errors.Annotatef(errors.ErrInvalidValue, "nil *Image")
			}
			return bot.AllocFromImage(img)

		case models.VolDataType:
			return bot.Alloc()

		default:
			return errors.Annotatef(errors.ErrInvalidValue, "invalid VolumeType: %s", vol.Type)
		}
	})
}

// Amplify .
func (vol *Volume) Amplify(delta int64, dom domain.Domain, ga agent.Interface, devPath string) (normDelta int64, err error) {
	normDelta = utils.NormalizeMultiple1024(delta)
	newCap := vol.Capacity + normDelta
	if newCap > configs.Conf.MaxVolumeCap {
		return 0, errors.Annotatef(errors.ErrInvalidValue, "exceeds the max cap: %d", configs.Conf.MaxVolumeCap)
	}

	least := utils.MaxInt64(
		configs.Conf.ResizeVolumeMinSize,
		int64(float64(vol.Capacity)*configs.Conf.ResizeVolumeMinRatio),
	)
	if least > normDelta {
		return 0, errors.Annotatef(errors.ErrInvalidValue, "invalid cap: at least %d, but %d",
			vol.Capacity+least, vol.Capacity+normDelta)
	}

	if err = vol.botOperate(func(bot Bot) error {
		return bot.Amplify(normDelta, dom, ga, devPath)
	}); err != nil {
		return 0, errors.Trace(err)
	}

	err = vol.Volume.Amplify(newCap)
	return
}

// Check .
func (vol *Volume) Check() error {
	return vol.botOperate(func(bot Bot) error {
		return bot.Check()
	})
}

// Repair .
func (vol *Volume) Repair() error {
	return vol.botOperate(func(bot Bot) error {
		return bot.Repair()
	})
}

// CreateSnapshot .
func (vol *Volume) CreateSnapshot() error {

	snapmod := models.NewSnapShot(vol.ID)
	snapmod.GenerateID()

	if err := vol.botOperate(func(bot Bot) error {
		return bot.CreateSnapshot(snapmod)
	}); err != nil {
		return errors.Trace(err)
	}

	if err := vol.AppendSnaps(snapmod); err != nil {
		return errors.Trace(err)
	}

	return vol.Save()
}

// CommitSnapshot .
func (vol *Volume) CommitSnapshot(snapID string) error {

	snap, err := vol.Snaps.Find(snapID)
	if err != nil {
		return errors.Trace(err)
	}

	if snap == nil {
		return errors.Annotatef(errors.ErrInvalidValue, "invalid snapID: %s", snapID)
	}

	if err := vol.botOperate(func(bot Bot) error {
		return bot.CommitSnapshot(snap)
	}); err != nil {
		return errors.Trace(err)
	}

	return vol.Save()
}

// CommitSnapshotByDay Commit snapshots created `day` days ago.
func (vol *Volume) CommitSnapshotByDay(day int) error {

	date := time.Now().AddDate(0, 0, -day).Unix()

	targetIdx := -1
	for i, s := range vol.Snaps {
		if s.CreatedTime > date { // Find the first snapshot that is created later than x days before
			targetIdx = i
		}
	}

	// No need to commit if all snapshot created within x days
	if targetIdx == 0 {
		return nil
	}

	// If all snapshot create x days before, keep the last one
	if targetIdx == -1 {
		targetIdx = vol.Snaps.Len() - 1
	}

	if err := vol.botOperate(func(bot Bot) error {
		return bot.CommitSnapshot(vol.Snaps[targetIdx])
	}); err != nil {
		return errors.Trace(err)
	}

	return vol.Save()
}

// RestoreSnapshot .
func (vol *Volume) RestoreSnapshot(snapID string) error {

	snap, err := vol.Snaps.Find(snapID)
	if err != nil {
		return errors.Trace(err)
	}

	if snap == nil {
		return errors.Annotatef(errors.ErrInvalidValue, "invalid snapID: %s", snapID)
	}

	if err := vol.botOperate(func(bot Bot) error {
		return bot.RestoreSnapshot(snap)
	}); err != nil {
		return errors.Trace(err)
	}

	return vol.Save()
}

// Mount .
func (vol *Volume) Mount(ga agent.Interface, devPath string) error {
	return vol.botOperate(func(bot Bot) error {
		return bot.Mount(ga, devPath)
	})
}

func (vol *Volume) botOperate(fn func(Bot) error) error {
	bot, err := vol.newBot(vol)
	if err != nil {
		return errors.Trace(err)
	}

	defer bot.Close()

	return fn(bot)
}
