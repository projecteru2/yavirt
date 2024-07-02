package guest

import (
	"context"
	"testing"

	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent/mocks"
	volFact "github.com/projecteru2/yavirt/internal/volume/factory"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
)

func TestCopyFileRunning(t *testing.T) {
	guest, bot := newMockedGuest(t)
	defer bot.AssertExpectations(t)

	f := &mocks.File{}
	defer f.AssertExpectations(t)

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	f.On("Write", mock.Anything, mock.Anything).Return(1, nil)
	f.On("Close", mock.Anything).Return(nil).Once()
	bot.On("RemoveAll", mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("MakeDirectory", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	bot.On("OpenFile", mock.Anything, mock.Anything, mock.Anything).Return(f, nil).Once()
	content := make(chan []byte, 10)
	content <- []byte{'a', 'b', 'c'}
	close(content)
	err := guest.copyToGuestRunning(ctx, "/root/test", content, bot, true)
	assert.NilErr(t, err)

	bot.On("IsFolder", mock.Anything, mock.Anything).Return(true, nil).Once()
	assert.Equal(t, terrors.ErrFolderExists, guest.copyToGuestRunning(ctx, "/root/test", content, bot, false))
}

func TestCopyFileNotRunning(t *testing.T) {
	var guest, _ = newMockedGuest(t)

	_, gfx := volFact.NewMockedVolume()
	gfx.On("Remove", mock.Anything).Return(nil).Once()
	gfx.On("MakeDirectory", mock.Anything, mock.Anything).Return(nil).Once()
	gfx.On("Upload", mock.Anything, mock.Anything).Return(nil).Once()

	content := make(chan []byte, 10)
	content <- []byte{'a', 'b', 'c'}
	close(content)
	assert.NilErr(t, guest.copyToGuestNotRunning("/root/test", content, true, gfx))

	gfx.On("IsDir", mock.Anything).Return(true, nil).Once()
	assert.Equal(t, terrors.ErrFolderExists, guest.copyToGuestNotRunning("/root/test", content, false, gfx))
}
