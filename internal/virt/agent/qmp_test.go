package agent

import (
	"context"
	"testing"

	"github.com/projecteru2/yavirt/pkg/libvirt/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func newMockQmp(dom *mocks.Domain) *qmp {
	return &qmp{
		name: "mock",
		virt: nil,
		ga:   true,
		dom:  dom,
	}
}

func TestFsFreezeAll(t *testing.T) {
	dom := &mocks.Domain{}
	q := newMockQmp(dom)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := `{"execute":"guest-fsfreeze-freeze"}`
	dom.On("QemuAgentCommand", ctx, cmd).Return(`{"return": 3}`, nil)
	nFs, err := q.FSFreezeAll(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 3, nFs)
}

func TestFSThawAll(t *testing.T) {
	dom := &mocks.Domain{}
	q := newMockQmp(dom)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := `{"execute":"guest-fsfreeze-thaw"}`
	dom.On("QemuAgentCommand", ctx, cmd).Return(`{"return": 3}`, nil)
	nFs, err := q.FSThawAll(ctx)
	assert.Nil(t, err)
	assert.Equal(t, 3, nFs)
}

func TestFsFreezeStatus(t *testing.T) {
	dom := &mocks.Domain{}
	q := newMockQmp(dom)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := `{"execute":"guest-fsfreeze-status"}`
	dom.On("QemuAgentCommand", ctx, cmd).Return(`{"return": "freezed"}`, nil)
	status, err := q.FSFreezeStatus(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "freezed", status)
}
