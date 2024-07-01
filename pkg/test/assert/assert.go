package assert

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cockroachdb/errors"
)

// NilErr .
func NilErr(t *testing.T, err error) {
	Nil(t, err, errors.GetReportableStackTrace(err))
}

// Err .
func Err(t *testing.T, err error) {
	NotNil(t, err, errors.GetReportableStackTrace(err))
}

// Nil .
func Nil(t *testing.T, obj any, msgAndArgs ...any) {
	require.Nil(t, obj, msgAndArgs...)
}

// NotNil .
func NotNil(t *testing.T, obj any, msgAndArgs ...any) {
	require.NotNil(t, obj, msgAndArgs...)
}

// True .
func True(t *testing.T, b bool, msgAndArgs ...any) {
	Equal(t, true, b, msgAndArgs...)
}

// False .
func False(t *testing.T, b bool, msgAndArgs ...any) {
	Equal(t, false, b, msgAndArgs...)
}

// Equal .
func Equal(t *testing.T, exp, act any, msgAndArgs ...any) {
	require.Equal(t, exp, act, msgAndArgs...)
}

// Fail .
func Fail(t *testing.T, fmt string, args ...any) {
	require.Fail(t, fmt, args...)
}
