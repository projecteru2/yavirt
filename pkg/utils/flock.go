package utils

import (
	"context"
	"os"
	"sync"
	"syscall"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/pkg/terrors"
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
		return errors.Wrap(err, "")
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
		return errors.WithMessagef(terrors.ErrFlockLocked, "%s has locked yet", f.fpth)
	}

	var stat, err = f.file.Stat()
	if err != nil {
		return errors.Wrap(err, "")
	}

	if stat.Mode()&0600 == 0600 {
		if err := f.reopen(); err != nil {
			return errors.Wrap(err, "")
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
		return errors.Wrap(err, "")
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
		log.Error(context.TODO(), err)
	}

	if err := os.Remove(f.fpth); err != nil {
		log.Error(context.TODO(), err)
	}
}

// Unlock .
func (f *Flock) Unlock() {
	f.mut.Lock()
	defer f.mut.Unlock()

	if err := f.unlock(); err != nil {
		log.Error(context.TODO(), err)
	}
}

func (f *Flock) RemoveFile() error {
	f.mut.Lock()
	defer f.mut.Unlock()
	err := os.Remove(f.fpth)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func (f *Flock) FileExists() bool {
	_, err := os.Stat(f.fpth)
	return err == nil
}

func (f *Flock) unlock() error {
	if err := syscall.Flock(int(f.file.Fd()), syscall.LOCK_UN); err != nil {
		return errors.Wrap(err, "")
	}

	f.locked = false

	f.close()

	return nil
}

func (f *Flock) close() {
	if err := f.file.Close(); err != nil {
		log.Warnf(context.TODO(), "[util] close %s failed", f.fpth)
	}
	f.file = nil
}
