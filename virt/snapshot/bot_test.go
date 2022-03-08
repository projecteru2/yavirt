package snapshot

import (
	"fmt"
	"testing"

	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/sh"
	shmocks "github.com/projecteru2/yavirt/pkg/sh/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
)

func TestBotCreate(t *testing.T) {
	shx := &shmocks.Shell{}
	defer shx.AssertExpectations(t)
	cancel := sh.NewMockShell(shx)
	defer cancel()

	shx.On("Copy", mock.Anything, mock.Anything).Return(nil).Once()
	shx.On("Move", mock.Anything, mock.Anything).Return(nil).Once()
	shx.On("Exec", mock.Anything, "qemu-img", "create",
		"-f", "qcow2", "-F", "qcow2", mock.Anything, "-b", mock.Anything).Return(nil).Once()
	shx.On("Exec", mock.Anything, "qemu-img", "rebase", "-b", mock.Anything, mock.Anything).Return(nil).Once()

	sbot := &bot{
		snap: &Snapshot{
			Snapshot: model.NewSnapShot("vol-id-test"),
		},
	}

	err := sbot.Create(model.NewVolume("vtype", 1))
	assert.NilErr(t, err)
}

func TestBotCommit(t *testing.T) {
	shx := &shmocks.Shell{}
	defer shx.AssertExpectations(t)
	cancel := sh.NewMockShell(shx)
	defer cancel()

	sbot := &bot{
		snap: &Snapshot{
			Snapshot: model.NewSnapShot("vol-id-test"),
		},
	}

	n := 5
	chain := generateMockedSnapshots(n)
	shx.On("Move", mock.Anything, mock.Anything).Return(nil).Once()
	shx.On("Remove", mock.Anything).Return(nil).Times(n - 1)
	shx.On("Exec", mock.Anything, "qemu-img", "commit", mock.Anything).Return(nil).Times(n - 1)

	err := sbot.Commit(chain)
	assert.NilErr(t, err)

	n = 1
	chain = generateMockedSnapshots(n)

	err = sbot.Commit(chain)
	assert.NilErr(t, err)
}

func TestBotRestore(t *testing.T) {
	shx := &shmocks.Shell{}
	defer shx.AssertExpectations(t)
	cancel := sh.NewMockShell(shx)
	defer cancel()

	sbot := &bot{
		snap: &Snapshot{
			Snapshot: model.NewSnapShot("vol-id-test"),
		},
	}

	n := 5
	chain := generateMockedSnapshots(n)
	shx.On("Move", mock.Anything, mock.Anything).Return(nil).Times(n + 1)
	shx.On("Remove", mock.Anything).Return(nil).Times(n - 1)
	shx.On("Copy", mock.Anything, mock.Anything).Return(nil).Times(n)
	shx.On("Exec", mock.Anything, "qemu-img", "commit", mock.Anything).Return(nil).Times(n - 1)

	err := sbot.Restore(model.NewVolume("vtype", 1), chain)
	assert.NilErr(t, err)

	n = 1
	chain = generateMockedSnapshots(n)
	shx.On("Move", mock.Anything, mock.Anything).Return(nil).Times(n + 1)
	shx.On("Remove", mock.Anything).Return(nil).Times(n - 1)
	shx.On("Copy", mock.Anything, mock.Anything).Return(nil).Times(n)
	shx.On("Exec", mock.Anything, "qemu-img", "commit", mock.Anything).Return(nil).Times(n - 1)

	err = sbot.Restore(model.NewVolume("vtype", 1), chain)
	assert.NilErr(t, err)
}

func TestBotUpload(t *testing.T) {
	// TODO
}

func TestBotDownload(t *testing.T) {
	// TODO
}

func TestBotDeleteFromBackupStorage(t *testing.T) {
	// TODO
}

func generateMockedSnapshots(length int) model.Snapshots {
	var chain model.Snapshots
	for i := 0; i < length; i++ {
		snap := model.NewSnapShot("vol-id-test")
		snap.ID = fmt.Sprintf("id-%d", i)
		chain = append(chain, snap)
	}
	return chain
}
