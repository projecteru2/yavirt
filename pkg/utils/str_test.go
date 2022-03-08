package utils

import (
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestMergeStrings(t *testing.T) {
	var cases = []struct {
		a, b []string
		exp  []string
	}{
		{
			[]string{},
			[]string{},
			[]string{},
		},
		{
			[]string{},
			[]string{"a"},
			[]string{"a"},
		},
		{
			[]string{"a"},
			[]string{},
			[]string{"a"},
		},
		{
			[]string{"a"},
			[]string{"b"},
			[]string{"a", "b"},
		},
		{
			[]string{"a", "a"},
			[]string{"b", "b"},
			[]string{"a", "b"},
		},
		{
			[]string{"a", "b"},
			[]string{"b", "c"},
			[]string{"a", "b", "c"},
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.exp, MergeStrings(c.a, c.b))
	}
}

func TestPartLeft(t *testing.T) {
	tests := []struct {
		in  string
		exp []string
	}{
		{"abc", []string{"abc", ""}},
		{".abc", []string{"", "abc"}},
		{"a.bc", []string{"a", "bc"}},
		{"abc.", []string{"abc", ""}},
		{"a.b.c", []string{"a", "b.c"}},
	}

	for _, tc := range tests {
		var a, b = PartLeft(tc.in, ".")
		assert.Equal(t, tc.exp[0], a)
		assert.Equal(t, tc.exp[1], b)
	}

}

func TestPartRight(t *testing.T) {
	var tests = []struct {
		in  string
		exp []string
	}{
		{"abc", []string{"", "abc"}},
		{".abc", []string{"", "abc"}},
		{"a.bc", []string{"a", "bc"}},
		{"abc.", []string{"abc", ""}},
		{"a.b.c", []string{"a.b", "c"}},
	}

	for _, tc := range tests {
		var a, b = PartRight(tc.in, ".")
		assert.Equal(t, tc.exp[0], a)
		assert.Equal(t, tc.exp[1], b)
	}
}
