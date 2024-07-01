package base

type Option func(o *OptionValue)
type OptionValue struct {
	Snapshot string
}

func WithSnapshot(snapshot string) Option {
	return func(o *OptionValue) {
		o.Snapshot = snapshot
	}
}
