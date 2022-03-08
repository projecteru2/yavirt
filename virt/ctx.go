package virt

import (
	"context"

	"github.com/projecteru2/yavirt/pkg/errors"
	calihandler "github.com/projecteru2/yavirt/vnet/handler/calico"
)

type key string

const calicoHandlerKey key = "CalicoHandler"

// Context .
type Context struct {
	context.Context
}

// NewContext .
func NewContext(ctx context.Context, caliHandler *calihandler.Handler) Context {
	ctx = context.WithValue(ctx, calicoHandlerKey, caliHandler)
	return Context{Context: ctx}
}

// CalicoHandler .
func (c Context) CalicoHandler() (*calihandler.Handler, error) {
	switch hand, ok := c.Value(calicoHandlerKey).(*calihandler.Handler); {
	case !ok:
		fallthrough
	case hand == nil:
		return nil, errors.Annotatef(errors.ErrInvalidValue, "nil *calihandler.Handler")

	default:
		return hand, nil
	}
}
