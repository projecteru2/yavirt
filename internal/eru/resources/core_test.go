package resources

import (
	"testing"

	cpumemtypes "github.com/projecteru2/core/resource/plugins/cpumem/types"
	"github.com/stretchr/testify/assert"
)

func TestConvCpumemBytes(t *testing.T) {
	testCases := []struct {
		name       string
		localNR    *cpumemtypes.NodeResource
		expected   []byte
		expectFail bool
	}{
		{
			name: "Valid case",
			localNR: &cpumemtypes.NodeResource{
				CPU:        4,
				Memory:     8192,
				NUMA:       map[string]string{"0": "0", "1": "0", "2": "1", "3": "1"},
				NUMAMemory: map[string]int64{"0": 1000, "1": 10000},
			},
			expected:   []byte(`{"cpu":4,"memory":6553,"numa-cpu":["0,1","2,3"],"numa-memory":["800","8000"]}`),
			expectFail: false,
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := convCpumemBytes(tc.localNR)
			if tc.expectFail {
				assert.Error(t, err, "Expected an error but got nil")
			} else {
				assert.NoError(t, err, "Expected no error but got %v", err)
				assert.Equal(t, tc.expected, result, "Result does not match expected")
			}
		})
	}
}
