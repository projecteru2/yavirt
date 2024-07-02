package guest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/cmd/run"
	intertypes "github.com/projecteru2/yavirt/internal/types"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

type buffer struct {
	sync.Mutex
	fromQ *utils.BytesQueue

	to   chan []byte
	quit chan struct{}
}

func (b *buffer) Close() error {
	close(b.to)
	close(b.quit)
	return nil
}

func (b *buffer) Read(p []byte) (int, error) {
	return b.fromQ.Read(p)
}

func (b *buffer) Write(p []byte) (int, error) {
	b.to <- bytes.Clone(p)
	return len(p), nil
}

func (b *buffer) UserRead() ([]byte, error) {
	bs, ok := <-b.to
	if !ok {
		return nil, io.EOF
	}
	return bs, nil
}

func (b *buffer) UserWrite(bs []byte) error {
	_, err := b.fromQ.Write(bs)
	return err
}

func attachGuest(c *cli.Context, runtime run.Runtime) error { //nolint
	id := c.Args().First()
	cmds := c.Args().Tail()
	timeout := c.Int("timeout")
	force := c.Bool("force")
	safe := c.Bool("safe")
	devname := c.String("devname")

	log.Debugf(c.Context, "attaching guest %s timeout %d", id, timeout)

	flags := intertypes.NewOpenConsoleFlags(force, safe, cmds)
	flags.Devname = devname
	stream := &buffer{
		fromQ: utils.NewBytesQueue(),
		to:    make(chan []byte, 10),
	}

	ctx, cancel := context.WithCancel(context.TODO())
	var lck sync.Mutex
	lastActive := time.Now()
	needExit := func() bool {
		lck.Lock()
		defer lck.Unlock()

		now := time.Now()
		elapse := now.Sub(lastActive)
		if elapse.Seconds() > float64(timeout) {
			cancel()
			return true
		}
		lastActive = now
		return false
	}
	go func() {
		err := runtime.Svc.AttachGuest(ctx, id, stream, flags)
		if err != nil {
			log.Errorf(c.Context, err, "attach guest error")
		}
	}()

	log.Debugf(c.Context, "start terminal...")

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint

	done1, done2 := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done1)
		defer log.Debugf(c.Context, "stdin done\n")

		buf := make([]byte, 100*1024)
		for {
			if needExit() {
				return
			}
			n, err := os.Stdin.Read(buf)
			if err != nil {
				fmt.Printf("Stdin: %s\n", err)
				return
			}
			bs := bytes.Clone(buf[:n])
			// find ^]
			if bytes.Contains(bs, []byte{uint8(29)}) {
				return
			}
			err = stream.UserWrite(bs)
			if err != nil {
				fmt.Printf("Stdin(Stream): %s\n", err)
				return
			}
		}
	}()
	go func() {
		defer close(done2)
		defer fmt.Printf("stdout done\n")
		for {
			if needExit() {
				return
			}
			bs, err := stream.UserRead()
			if err != nil {
				fmt.Printf("Stdout(Stream): %s\n", err)
				return
			}
			log.Debugf(c.Context, "[Exec:Stdout] got from stream: %v\r\n", bs)
			_, err = os.Stdout.Write(bs)
			if err != nil {
				fmt.Printf("Stdout: %s\n", err)
				return
			}
		}
	}()

	select {
	case <-done1:
	case <-done2:
	}
	return nil
}
