package snapshot

import (
	"testing"

	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	snapmock "github.com/projecteru2/yavirt/internal/virt/snapshot/mocks"
)

func TestCreate(t *testing.T) {
	snapmod := model.NewSnapShot("vol-id-123")

	sbot := &snapmock.Bot{}
	sbot.On("Create", mock.Anything).Return(nil).Once()
	sbot.On("Close").Return(nil).Twice()
	sbot.On("Upload", mock.Anything).Return(nil).Once()

	snap := &Snapshot{
		Snapshot: snapmod,
		newBot:   func(v *Snapshot) (Bot, error) { return sbot, nil },
	}

	err := snap.Create(model.NewVolume("vol-id-123", 1))
	assert.NilErr(t, err)
}

func TestCommit(t *testing.T) {
	snapmod := model.NewSnapShot("vol-id")
	snapmod.ID = "id-8"
	snapmod.BaseSnapshotID = "id-4"

	sbot := &snapmock.Bot{}
	sbot.On("Commit", mock.Anything).Return(nil).Once()
	sbot.On("Close").Return(nil).Times(10 + 2)
	sbot.On("Download", mock.Anything).Return(nil).Times(10)
	sbot.On("Upload", mock.Anything).Return(nil).Once()

	snap := &Snapshot{
		Snapshot: snapmod,
		newBot:   func(v *Snapshot) (Bot, error) { return sbot, nil },
	}

	snaps := generateMockedSnapshots(10)
	snaps[8].BaseSnapshotID = "id-4"
	snaps[4].BaseSnapshotID = "id-2"
	snaps[2].BaseSnapshotID = "id-1"

	ret, err := snap.Commit(snaps)
	assert.NilErr(t, err)
	assert.Equal(t, ret.Len(), 3)
	assert.Equal(t, ret[0], snaps[4])
	assert.Equal(t, ret[1], snaps[2])
	assert.Equal(t, ret[2], snaps[1])
}

func TestCommitRoot(t *testing.T) {
	snapmod := model.NewSnapShot("vol-id-123")
	snapmod.ID = "id-8"

	sbot := &snapmock.Bot{}
	sbot.On("Commit", mock.Anything).Return(nil).Once()
	sbot.On("Close").Return(nil).Times(10 + 2)
	sbot.On("Download", mock.Anything).Return(nil).Times(10)
	sbot.On("Upload", mock.Anything).Return(nil).Once()

	snap := &Snapshot{
		Snapshot: snapmod,
		newBot:   func(v *Snapshot) (Bot, error) { return sbot, nil },
	}

	snaps := generateMockedSnapshots(10)

	ret, err := snap.Commit(snaps)
	assert.NilErr(t, err)
	assert.Equal(t, ret.Len(), 0)
}

func TestRestore(t *testing.T) {
	snapmod := model.NewSnapShot("vol-id-123")
	snapmod.ID = "id-123"

	sbot := &snapmock.Bot{}
	sbot.On("Close").Return(nil).Times(10 + 1)
	sbot.On("Download", mock.Anything).Return(nil).Times(10)
	sbot.On("Restore", mock.Anything, mock.Anything).Return(nil).Once()

	snap := &Snapshot{
		Snapshot: snapmod,
		newBot:   func(v *Snapshot) (Bot, error) { return sbot, nil },
	}

	snaps := generateMockedSnapshots(3)
	snaps[0].ID = "id-123"

	err := snap.Restore(model.NewVolume("vol-id-123", 1), snaps)
	assert.NilErr(t, err)
}
