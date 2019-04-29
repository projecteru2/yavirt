package errors

import je "github.com/juju/errors"

var (
	// New .
	New = je.New
	// Stack .
	Stack = je.ErrorStack
	// Trace .
	Trace = je.Trace
	// Annotatef .
	Annotatef = je.Annotatef
	// Errorf .
	Errorf = je.Errorf
	// Wrap .
	Wrap = je.Wrap
	// Cause .
	Cause = je.Cause
)

// Contain .
func Contain(a, b error) bool {
	var pre = Cause(a)

	// No previous error.
	if pre == a {
		return a == b
	}

	if pre == b {
		return true
	}

	return Contain(pre, b)
}
