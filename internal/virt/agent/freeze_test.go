package agent

import (
	"context"
	"testing"

	"github.com/projecteru2/yavirt/internal/virt/agent/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestFreeze(t *testing.T) {
	mockQmp := mocks.NewQmp(t)
	ag := Agent{
		qmp: mockQmp,
	}

	mockQmp.On("FSFreezeAll", context.Background()).Return(1, nil).Once()
	nFS, err := ag.FSFreezeAll(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, nFS)

	mockQmp.On("FSThawAll", context.Background()).Return(1, nil).Once()
	nFS, err = ag.FSThawAll(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, 1, nFS)

	mockQmp.On("FSFreezeStatus", context.Background()).Return("freezed", nil).Once()
	status, err := ag.FSFreezeStatus(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, "freezed", status)
}
