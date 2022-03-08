package guest

import (
	"syscall"
)

type fdAdapter struct {
	fd int
}

func (a fdAdapter) Read(p []byte) (n int, err error) {
	n, err = syscall.Read(a.fd, p)
	if err != nil { // Linux syscall.Read may return n = -1 on error
		n = 0
	}
	return
}

func (a fdAdapter) Write(p []byte) (n int, err error) {
	n, err = syscall.Write(a.fd, p)
	if err != nil { // Linux syscall.Write may return n = -1 on error
		n = 0
	}
	return
}

func (a fdAdapter) Fd() int {
	return a.fd
}

func (a fdAdapter) Close() error {
	return syscall.Close(a.fd)
}
