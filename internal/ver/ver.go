package ver

import "fmt"

var (
	// Git commit
	Git string
	// Compile info. of golang itself.
	Compile string
	// Date of compiled
	Date string
)

// Version .
func Version() string {
	return fmt.Sprintf(`Git: %s
Compile: %s
Built: %s`, Git, Compile, Date)
}
