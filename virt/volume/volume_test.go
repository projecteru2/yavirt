package volume

import (
	"testing"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	storemocks "github.com/projecteru2/yavirt/pkg/store/mocks"
	"github.com/projecteru2/yavirt/test/assert"
	"github.com/projecteru2/yavirt/test/mock"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/virt/volume/mocks"
)

func TestAmplifyFailed_DeltaLessThanMinSize(t *testing.T) {
	volmod, err := model.NewDataVolume("/data", util.TB)
	assert.NilErr(t, err)

	vol := &Volume{Volume: volmod}

	cap := vol.Capacity + config.Conf.ResizeVolumeMinSize - 1
	delta, err := vol.Amplify(cap, nil, nil, "")
	assert.Err(t, err)
	assert.Equal(t, int64(0), delta)
}

func TestAmplifyFailed_DeltaLessThanMinRatio(t *testing.T) {
	volmod, err := model.NewDataVolume("/data", util.TB)
	assert.NilErr(t, err)

	vol := &Volume{Volume: volmod}

	cap := vol.Capacity + int64(float64(vol.Capacity)*config.Conf.ResizeVolumeMinRatio-1)
	delta, err := vol.Amplify(cap, nil, nil, "")
	assert.Err(t, err)
	assert.Equal(t, int64(0), delta)
}

func TestAmplify(t *testing.T) {
	volmod, err := model.NewDataVolume("/data", 10*util.GB)
	assert.NilErr(t, err)

	bot := &mocks.Bot{}
	defer bot.AssertExpectations(t)

	vol := &Volume{
		Volume: volmod,
		newBot: func(v *Volume) (Bot, error) { return bot, nil },
	}

	meta, metaCancel := storemocks.Mock()
	defer metaCancel()
	defer meta.AssertExpectations(t)
	meta.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	bot.On("Close").Return(nil).Once()
	bot.On("Amplify", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	delta, err := vol.Amplify(config.Conf.ResizeVolumeMinSize, nil, nil, "")
	assert.Nil(t, err)
	assert.Equal(t, config.Conf.ResizeVolumeMinSize, delta)
	assert.Equal(t, 20*util.GB, vol.Capacity)
}

func TestAttachVolume_Rollback(t *testing.T) {
}

func TestAttachVolume(t *testing.T) {
	// TODO
}
