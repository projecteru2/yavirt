package model

import (
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestStatusForward(t *testing.T) {
	var cases = []struct {
		forward string
		allowed map[string]struct{}
	}{
		{
			StatusPending,
			allow([]string{StatusPending, ""}),
		},
		{
			StatusCreating,
			allow([]string{StatusCreating, StatusPending}),
		},
		{
			StatusStarting,
			allow([]string{StatusStarting, StatusCreating, StatusStopped}),
		},
		{
			StatusRunning,
			allow([]string{StatusRunning, StatusStarting}),
		},
		{
			StatusStopping,
			allow([]string{StatusStopping, StatusRunning, StatusStopped}),
		},
		{
			StatusStopped,
			allow([]string{StatusStopped, StatusStopping, StatusMigrating, StatusCaptured}),
		},
		{
			StatusMigrating,
			allow([]string{StatusMigrating, StatusStopped, StatusResizing}),
		},
		{
			StatusResizing,
			allow([]string{StatusResizing, StatusStopped, StatusRunning}),
		},
		{
			StatusDestroying,
			allow([]string{StatusDestroying, StatusStopped, StatusDestroyed}),
		},
		{
			StatusDestroyed,
			allow([]string{StatusDestroyed, StatusDestroying}),
		},
		{
			StatusCapturing,
			allow([]string{StatusCapturing, StatusStopped}),
		},
		{
			StatusCaptured,
			allow([]string{StatusCaptured, StatusCapturing}),
		},
	}

	var g = newGeneric()

	for _, c := range cases {
		var next = c.forward
		for _, now := range AllStatuses {
			g.Status = now

			if _, exists := c.allowed[now]; exists {
				assert.True(t, g.checkStatus(next), "expect true of %s to %s", now, next)
			} else {
				assert.False(t, g.checkStatus(next), "expect false of %s to %s", now, next)
			}
		}
	}
}

func allow(st []string) map[string]struct{} {
	var m = map[string]struct{}{}

	for _, elem := range st {
		m[elem] = struct{}{}
	}

	return m
}
