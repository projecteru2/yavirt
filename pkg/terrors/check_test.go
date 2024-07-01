package terrors

import (
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsVirtLinkRouteExistsErr(t *testing.T) {
	err := ErrVirtLinkRouteExists
	err = errors.Wrap(err, "test")
	assert.True(t, IsVirtLinkRouteExistsErr(err))
	err = errors.WithMessage(err, "test1")
	assert.True(t, IsVirtLinkRouteExistsErr(err))
}

// func TestWrap(t *testing.T) {
// 	err1 := func() error {
// 		return func() error {
// 			return errors.Wrap(ErrVirtLinkRouteExists, "test")
// 		}()
// 	}()
// 	err2 := func() error {
// 		err := func() error {
// 			return errors.Wrap(ErrVirtLinkRouteExists, "test")
// 		}()
// 		return errors.Wrap(err, "test1")
// 	}()
// 	assert.Truef(t, false, "error1: %+v", err1)
// 	assert.Truef(t, false, "error2: %+v", err2)
// 	assert.Truef(t, false, "error3: %+v", errors.WithMessage(err1, "test3"))
// }
