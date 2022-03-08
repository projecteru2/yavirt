package models

import (
	"fmt"
	"strconv"
	"testing"

	erucluster "github.com/projecteru2/core/cluster"
	erutypes "github.com/projecteru2/core/types"
	eruutils "github.com/projecteru2/core/utils"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func TestEmptyLabels(t *testing.T) {
	g := Guest{JSONLabels: map[string]string{}}

	_, err := g.PublishPorts()
	assert.Err(t, err)

	_, err = g.HealthCheck()
	assert.Err(t, err)
}

func TestValidLabels(t *testing.T) {
	g := Guest{
		JSONLabels: map[string]string{
			erucluster.LabelMeta: eruutils.EncodeMetaInLabel(&erutypes.LabelMeta{
				Publish: []string{"22", "80"},
				HealthCheck: &erutypes.HealthCheck{
					TCPPorts: []string{"2379", "3306"},
					HTTPPort: "80",
					HTTPURL:  "/check",
					HTTPCode: 200,
				},
			}),
			"unknown": "unknown",
		},
	}

	ports, err := g.PublishPorts()
	assert.NilErr(t, err)
	assert.Equal(t, []int{22, 80}, ports)

	hc, err := g.HealthCheck()
	assert.Equal(t, []int{2379, 3306}, hc.TCPPorts)
	assert.Equal(t, 80, hc.HTTPPort)
	assert.Equal(t, "/check", hc.HTTPPath)
	assert.Equal(t, 200, hc.HTTPCode)
}

func TestRemoveVol(t *testing.T) {
	testcases := []struct {
		orig []int
		rm   int
		ids  []int
	}{
		// removes the first item.
		{
			[]int{0},
			0,
			[]int{},
		},
		{
			[]int{0, 1},
			0,
			[]int{1},
		},
		{
			[]int{0, 1, 2},
			0,
			[]int{2, 1},
		},
		{
			[]int{0, 1, 2, 3},
			0,
			[]int{3, 1, 2},
		},
		// removes the last item.
		{
			[]int{0, 1},
			1,
			[]int{0},
		},
		{
			[]int{0, 1, 2},
			2,
			[]int{0, 1},
		},
		{
			[]int{0, 1, 2, 3},
			3,
			[]int{0, 1, 2},
		},
		// removes the medium item.
		{
			[]int{0, 1, 2},
			1,
			[]int{0, 2},
		},
		{
			[]int{0, 1, 2, 3},
			2,
			[]int{0, 1, 3},
		},
		{
			[]int{0, 1, 2, 3},
			1,
			[]int{0, 3, 2},
		},
		// duplicated
		{
			[]int{0, 0, 0},
			0,
			[]int{},
		},
		{
			[]int{0, 0, 1},
			0,
			[]int{1},
		},
		{
			[]int{0, 1, 1},
			1,
			[]int{0},
		},
		{
			[]int{0, 1, 0, 1},
			0,
			[]int{1, 1},
		},
		{
			[]int{0, 1, 0, 1},
			1,
			[]int{0, 0},
		},
		{
			[]int{0, 1, 1, 0},
			0,
			[]int{1, 1},
		},
		{
			[]int{0, 1, 1, 0},
			1,
			[]int{0, 0},
		},
	}

	for _, tc := range testcases {
		g := newGuest()
		for _, id := range tc.orig {
			vol, err := NewDataVolume(fmt.Sprintf("/data%d", id), utils.GB)
			assert.NilErr(t, err)

			vol.ID = strconv.Itoa(id)
			assert.NilErr(t, g.AppendVols(vol))
		}

		g.RemoveVol(strconv.Itoa(tc.rm))
		assert.Equal(t, len(tc.ids), g.Vols.Len())
		assert.Equal(t, len(tc.ids), len(g.VolIDs))

		for i, id := range tc.ids {
			assert.Equal(t, strconv.Itoa(id), g.Vols[i].ID)
			assert.Equal(t, strconv.Itoa(id), g.VolIDs[i])
		}
	}
}
