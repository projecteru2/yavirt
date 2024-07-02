package types

import (
	"io"

	"github.com/projecteru2/yavirt/pkg/libvirt"
)

type Console interface {
	io.ReadWriteCloser
	Fd() int // need fd for epoll event
}

// OpenConsoleFlags .
type OpenConsoleFlags struct {
	libvirt.ConsoleFlags
	Devname  string
	Commands []string
}

// NewOpenConsoleFlags .
func NewOpenConsoleFlags(force, safe bool, cmds []string) OpenConsoleFlags {
	return OpenConsoleFlags{
		ConsoleFlags: libvirt.ConsoleFlags{
			Force: force,
			Safe:  safe,
		},
		Commands: cmds,
	}
}
