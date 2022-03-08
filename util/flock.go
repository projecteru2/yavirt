package util

import (
	"os"
	"sync"
	"syscall"

	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/log"
)

const perm = os.FileMode(0600)

// Flock .
type Flock struct {
	mut    sync.Mutex
	fpth   string
	file   *os.File
	locked bool
}

// NewFlock .
func NewFlock(fpth string) *Flock {
	return &Flock{fpth: fpth}
}

// Trylock .
func (f *Flock) Trylock() error {
	f.mut.Lock()
	defer f.mut.Unlock()

	if f.locked {
		return nil
	}

	if err := f.open(); err != nil {
		return errors.Trace(err)
	}

	return f.flock(true)
}

func (f *Flock) flock(retry bool) error {
	switch err := syscall.Flock(int(f.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); {
	case err == nil:
		f.locked = true
		return nil

	case err == syscall.EWOULDBLOCK:
		fallthrough
	case err != syscall.EIO && err != syscall.EBADF:
		fallthrough
	case !retry:
		return errors.Errorf("%s has locked yet", f.fpth)
	}

	var stat, err = f.file.Stat()
	if err != nil {
		return errors.Trace(err)
	}

	if stat.Mode()&0600 == 0600 { //nolint
		if err := f.reopen(); err != nil {
			return errors.Trace(err)
		}
	}

	return f.flock(false)
}

func (f *Flock) reopen() error {
	f.close()
	return f.open()
}

func (f *Flock) open() error {
	if f.file != nil {
		return nil
	}

	var fh, err = os.OpenFile(f.fpth, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		return errors.Trace(err)
	}

	f.file = fh

	return nil
}

// Close .
func (f *Flock) Close() {
	f.mut.Lock()
	defer f.mut.Unlock()

	if !f.locked {
		return
	}

	if err := f.unlock(); err != nil {
		log.WarnStack(err)
	}

	if err := os.Remove(f.fpth); err != nil {
		log.WarnStack(err)
	}
}

// Unlock .
func (f *Flock) Unlock() {
	f.mut.Lock()
	defer f.mut.Unlock()

	if err := f.unlock(); err != nil {
		log.WarnStack(err)
	}
}

func (f *Flock) unlock() error {
	if err := syscall.Flock(int(f.file.Fd()), syscall.LOCK_UN); err != nil {
		return errors.Trace(err)
	}

	f.locked = false

	f.close()

	return nil
}

func (f *Flock) close() {
	if err := f.file.Close(); err != nil {
		log.Warnf("[util] close %s failed", f.fpth)
	}
	f.file = nil
}
