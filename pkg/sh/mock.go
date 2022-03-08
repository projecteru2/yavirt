package sh

// NewMockShell .
func NewMockShell(s Shell) func() {
	var old = shell

	shell = s

	return func() {
		shell = old
	}
}
