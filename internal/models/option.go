package models

type Op struct {
	IgnoreLoadImageErr bool
}

type Option func(*Op)

func IgnoreLoadImageErrOption() Option {
	return func(op *Op) {
		op.IgnoreLoadImageErr = true
	}
}

func NewOp(opts ...Option) *Op {
	op := &Op{}
	for _, opt := range opts {
		opt(op)
	}
	return op
}
