package guest

import (
	"context"
	"testing"

	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/test/assert"
	"github.com/projecteru2/yavirt/test/mock"
	"github.com/projecteru2/yavirt/virt/agent/mocks"
	"github.com/projecteru2/yavirt/virt/volume"
)

func TestCopyFileRunning(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	f := &mocks.File{}
	defer f.AssertExpectations(t)

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	f.On("Write", mock.Anything).Return(1, nil)
	f.On("Close").Return(nil).Once()
	bot.On("RemoveAll", mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("MakeDirectory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("OpenFile", mock.Anything, mock.Anything).Return(f, nil).Once()
	content := make(chan []byte, 10)
	content <- []byte{'a', 'b', 'c'}
	close(content)
	assert.NilErr(t, guest.copyToGuestRunning(ctx, "/root/test", content, bot, true))

	bot.On("IsFolder", mock.Anything, mock.Anything).Return(true, nil).Once()
	assert.Equal(t, errors.ErrFolderExists, guest.copyToGuestRunning(ctx, "/root/test", content, bot, false))
}

func TestCopyFileNotRunning(t *testing.T) {
	var guest, _ = newMockedGuest(t)

	_, gfx := volume.NewMockedVolume()
	gfx.On("Remove", mock.Anything).Return(nil).Once()
	gfx.On("MakeDirectory", mock.Anything, mock.Anything).Return(nil).Once()
	gfx.On("Upload", mock.Anything, mock.Anything).Return(nil).Once()

	content := make(chan []byte, 10)
	content <- []byte{'a', 'b', 'c'}
	close(content)
	assert.NilErr(t, guest.copyToGuestNotRunning("/root/test", content, true, gfx))

	gfx.On("IsDir", mock.Anything).Return(true, nil).Once()
	assert.Equal(t, errors.ErrFolderExists, guest.copyToGuestNotRunning("/root/test", content, false, gfx))
}
