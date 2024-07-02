package guest

import (
	"context"
	"os"
	"testing"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent/mocks"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
)

func TestLogRunning(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	f := &mocks.File{}
	f.On("Close", mock.Anything).Return(nil)
	defer f.AssertExpectations(t)

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	content := []byte{'a', 'b', 'c'}

	f.On("CopyTo", mock.Anything, mock.Anything).Return(0, nil).Once()
	bot.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Return(f, nil).Once()
	tmp, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NilErr(t, err)
	err = guest.logRunning(ctx, bot, -1, "/tmp/log", tmp)
	assert.NilErr(t, err)
	tmp.Close()

	bot.On("OpenFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(f, nil).Once()
	f.On("Tail", mock.Anything, mock.Anything).Return(content, nil).Once()
	tmp, err = os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NilErr(t, err)
	err = guest.logRunning(ctx, bot, 1, "/tmp/log", tmp)
	assert.NilErr(t, err)
	tmp.Close()
}

func TestLogStopped(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	_, gfx := volFact.NewMockedVolume()
	defer gfx.AssertExpectations(t)

	content := []byte{'a', 'b', 'c'}

	tmp, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	gfx.On("Read", mock.Anything).Return(content, nil).Once()
	assert.NilErr(t, err)
	defer tmp.Close()
	assert.NilErr(t, guest.logStopped(-1, "/tmp/log", tmp, gfx))

	logs := []string{"a", "b"}
	gfx.On("Tail", mock.Anything, mock.Anything).Return(logs, nil).Once()
	assert.NilErr(t, guest.logStopped(1, "/tmp/log", tmp, gfx))
}
