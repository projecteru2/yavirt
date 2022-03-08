package assert

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projecteru2/yavirt/internal/errors"
)

// NilErr .
func NilErr(t *testing.T, err error) {
	Nil(t, err, errors.Stack(err))
}

// Err .
func Err(t *testing.T, err error) {
	NotNil(t, err, errors.Stack(err))
}

// Nil .
func Nil(t *testing.T, obj interface{}, msgAndArgs ...interface{}) {
	require.Nil(t, obj, msgAndArgs...)
}

// NotNil .
func NotNil(t *testing.T, obj interface{}, msgAndArgs ...interface{}) {
	require.NotNil(t, obj, msgAndArgs...)
}

// True .
func True(t *testing.T, b bool, msgAndArgs ...interface{}) {
	Equal(t, true, b, msgAndArgs...)
}

// False .
func False(t *testing.T, b bool, msgAndArgs ...interface{}) {
	Equal(t, false, b, msgAndArgs...)
}

// Equal .
func Equal(t *testing.T, exp, act interface{}, msgAndArgs ...interface{}) {
	require.Equal(t, exp, act, msgAndArgs...)
}

// Fail .
func Fail(t *testing.T, fmt string, args ...interface{}) {
	require.Fail(t, fmt, args...)
}
