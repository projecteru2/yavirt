package libvirt

import (
	"context"
	"io"
	"sync"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/pkg/utils"
	libvirtgo "github.com/projecteru2/yavirt/third_party/libvirt"
)

type ConsoleFlags struct {
	Force    bool
	Safe     bool
	Nonblock bool
}

func (cf *ConsoleFlags) genLibvirtFlags() (flags libvirtgo.DomainConsoleFlags) {
	if cf.Force {
		flags |= libvirtgo.DomainConsoleForce
	}
	if cf.Safe {
		flags |= libvirtgo.DomainConsoleSafe
	}
	return
}

func (cf *ConsoleFlags) genStreamFlags() (flags libvirtgo.StreamFlags) { //nolint
	if cf.Nonblock {
		flags = libvirtgo.StreamNonblock
	}
	return
}

type Console struct {
	// pty to user
	fromQ *utils.BytesQueue
	// user to pty
	toQ *utils.BytesQueue

	quit struct {
		once sync.Once
		c    chan struct{}
	}
}

func newConsole() *Console {
	con := &Console{
		fromQ: utils.NewBytesQueue(),
		toQ:   utils.NewBytesQueue(),
	}
	con.quit.c = make(chan struct{})
	return con
}

func (c *Console) needExit(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	case <-c.quit.c:
		return true
	default:
		return false
	}
}

func (c *Console) From(ctx context.Context, r io.Reader) error {
	logger := log.WithFunc("Console.From")
	buf := make([]byte, 64*1024)
	for {
		if c.needExit(ctx) {
			return nil
		}
		n, err := r.Read(buf)
		if n == 0 {
			if err != nil {
				if err != io.EOF {
					logger.Errorf(ctx, err, "[Console:From] read error")
				}
				return err
			}
			continue
		}

		bs := buf[:n]
		_, err = c.toQ.Write(bs)
		if err != nil {
			logger.Errorf(ctx, err, "write error")
			return err
		}
	}
}

func (c *Console) To(ctx context.Context, w io.Writer) error {
	logger := log.WithFunc("Console.To")
	buf := make([]byte, 64*1024)
	for {
		if c.needExit(ctx) {
			return nil
		}
		// pty to user
		n, err := c.fromQ.Read(buf)
		if n == 0 {
			if err != nil {
				if err != io.EOF {
					logger.Errorf(ctx, err, "read error")
				}
				return err
			}
			continue
		}
		if c.needExit(ctx) {
			return nil
		}

		_, err = w.Write(buf[:n])
		if err != nil {
			logger.Errorf(ctx, err, "write error")
			return err
		}
		copy(buf, make([]byte, len(buf)))
	}
}

func (c *Console) Write(buf []byte) (int, error) {
	return c.fromQ.Write(buf)
}

func (c *Console) Read(p []byte) (int, error) {
	return c.toQ.Read(p)
}

func (c *Console) Close() {
	c.quit.once.Do(func() {
		c.fromQ.Close()
		c.toQ.Close()
		close(c.quit.c)
	})
}
