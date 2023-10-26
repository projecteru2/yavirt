package libvirt

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/projecteru2/yavirt/pkg/log"
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

type Console struct {
	Stream *Stream
	// pty to user
	fromQ *utils.BytesQueue
	// user to pty
	toQ  *utils.BytesQueue
	quit quit
}

type quit struct {
	once sync.Once
	q    chan struct{}
}

func newConsole(s *Stream) *Console {
	return &Console{
		Stream: s,
		fromQ:  utils.NewBytesQueue(),
		toQ:    utils.NewBytesQueue(),
		quit: quit{
			once: sync.Once{},
			q:    make(chan struct{}),
		},
	}
}

func (c *Console) needExit() bool {
	select {
	case <-c.quit.q:
		return true
	default:
		return false
	}
}

func (c *Console) From(_ context.Context, r io.Reader) error {
	buf := make([]byte, 64*1024)
	for {
		if c.needExit() {
			return nil
		}
		// Read a single byte
		n, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorf("[Console:From] read error: %s", err)
			}
			return err
		}

		if n == 0 {
			continue
		}

		bs := buf[:n]
		cloneBs := bytes.Clone(bs)
		copy(buf, make([]byte, len(buf)))
		_, err = c.toQ.Write(cloneBs)
		if err != nil {
			log.Errorf("[Console:From] write error: %s", err)
			return err
		}
	}
}

func (c *Console) To(_ context.Context, w io.Writer) error {
	buf := make([]byte, 64*1024)
	for {
		if c.needExit() {
			return nil
		}
		// pty to user
		n, err := c.fromQ.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorf("[Console:To] read error: %s", err)
			}
			return err
		}
		if n == 0 {
			continue
		}
		if c.needExit() {
			return nil
		}

		_, err = w.Write(buf[:n])
		if err != nil {
			log.Errorf("[Console:To] write error: %s", err)
			return err
		}
		copy(buf, make([]byte, len(buf)))
	}
}

func (c *Console) GetInputToPtyReader() io.ReadWriter {
	return c.fromQ
}

func (c *Console) GetOutputToUserWriter() io.ReadWriter {
	return c.toQ
}

func (c *Console) Close() {
	c.quit.once.Do(func() {
		defer func() {
			close(c.quit.q)
		}()
		// c.Stream.EventRemoveCallback() //nolint
		c.Stream.Close()
		c.fromQ.Close()
		c.toQ.Close()
	})
}

func sendAll(stream *Stream, bs []byte) error {
	for len(bs) > 0 {
		// inStream
		n, err := stream.Send(bs)
		if err != nil {
			return err
		}
		bs = bs[n:]
	}
	return nil
}

// AddReadWriter For block stream IO
func (c *Console) AddReadWriter() error { //nolint
	go func() {
		defer log.Infof("[AddReadWriter] Send goroutine exit")
		for {
			if c.needExit() {
				return
			}
			// from user input, send to pty
			bs, err := c.toQ.Pop()
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console toQ queue: %s", err)
				return
			}
			if c.needExit() {
				return
			}
			err = sendAll(c.Stream, bs)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console stream: %s", err)
				return
			}
		}
	}()
	go func() {
		defer log.Infof("[AddReadWriter] Recv goroutine exit")
		buf := make([]byte, 100*1024)
		for {
			if c.needExit() {
				return
			}
			n, err := c.Stream.Recv(buf)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when read from console stream: %s", err)
				return
			}
			if n == 0 {
				continue
			}
			bs := buf[:n]
			if c.needExit() {
				return
			}
			_, err = c.fromQ.Write(bs)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console queue: %s", err)
				return
			}
			copy(buf, make([]byte, len(buf)))
		}
	}()
	return nil
}
