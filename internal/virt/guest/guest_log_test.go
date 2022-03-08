package guest

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	"github.com/projecteru2/yavirt/internal/virt/agent/mocks"
	"github.com/projecteru2/yavirt/internal/virt/volume"
)

func TestLogRunning(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	f := &mocks.File{}
	f.On("Read", mock.Anything).Return(0, io.EOF)
	f.On("Close").Return(nil)
	defer f.AssertExpectations(t)

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	content := []byte{'a', 'b', 'c'}

	bot.On("OpenFile", mock.Anything, mock.Anything).Return(f, nil).Once()
	tmp, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NilErr(t, err)
	assert.NilErr(t, guest.logRunning(ctx, bot, -1, "/tmp/log", tmp))
	tmp.Close()

	bot.On("OpenFile", mock.Anything, mock.Anything).Return(f, nil).Once()
	f.On("Tail", mock.Anything).Return(content, nil).Once()
	tmp, err = os.OpenFile("/dev/null", os.O_WRONLY, 0)
	assert.NilErr(t, err)
	assert.NilErr(t, guest.logRunning(ctx, bot, 1, "/tmp/log", tmp))
	tmp.Close()
}

func TestLogStopped(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	_, gfx := volume.NewMockedVolume()
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
