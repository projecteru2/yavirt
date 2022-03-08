package guest

import (
	"io"
	"os"
	"sync"

	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/virt/types"
	"golang.org/x/sys/unix"
)

// implement a epoll based console from guest's natural console
// ref. https://github.com/containerd/console

const (
	maxEvent = 128
)

// epoll fd and uses epoll API to perform I/O.
type EpollConsole struct {
	types.Console
	readc  *sync.Cond // signal read is ready upon receiving epoll event
	writec *sync.Cond // signal write is ready upon receiving epoll event
	sysfd  int
	closed bool
}

type Epoller struct {
	efd       int
	mu        sync.Mutex
	fdMapping map[int]*EpollConsole // to reverse look up epoll triggered console from event's fd
	closeOnce sync.Once             // only close once on calling Close()
}

var currentEpoller *Epoller

func SetupEpoller() error {
	epoller, err := NewEpoller()
	if err != nil {
		return errors.Annotatef(err, "failed to initialize epoller")
	}
	currentEpoller = epoller
	go epoller.Wait() // nolint
	return nil
}

func GetCurrentEpoller() *Epoller {
	return currentEpoller
}

func NewEpoller() (*Epoller, error) { // only need 1 epoller upon deamon start & have multiple console registered
	efd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return &Epoller{
		efd:       efd,
		fdMapping: make(map[int]*EpollConsole),
	}, nil
}

func (e *Epoller) Add(c types.Console) (*EpollConsole, error) {
	fd := c.Fd()
	// set console fd to non-blocking for epoll
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	ev := unix.EpollEvent{
		Events: unix.EPOLLIN | unix.EPOLLOUT | unix.EPOLLRDHUP | unix.EPOLLET,
		Fd:     int32(fd),
	}
	if err := unix.EpollCtl(e.efd, unix.EPOLL_CTL_ADD, fd, &ev); err != nil { // register into epoller
		return nil, err
	}
	ef := &EpollConsole{
		Console: c,
		sysfd:   fd,
		readc:   sync.NewCond(&sync.Mutex{}),
		writec:  sync.NewCond(&sync.Mutex{}),
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fdMapping[fd] = ef
	return ef, nil
}

// Wait to be run in a separate go routinue, converting epoll event to read/write Cond's singal
// https://man7.org/linux/man-pages/man2/epoll_wait.2.html
func (e *Epoller) Wait() error {
	events := make([]unix.EpollEvent, maxEvent)
	for {
		n, err := unix.EpollWait(e.efd, events, -1) // n: # of events received
		if err != nil {
			if err == unix.EINTR { // interrupted or timeout
				continue
			}
			return err
		}
		for i := 0; i < n; i++ {
			ev := &events[i]
			// EPOLLIN: read; EPOLLHUP/EPOLLERR: read close connection or error
			if ev.Events&(unix.EPOLLIN|unix.EPOLLHUP|unix.EPOLLERR) != 0 {
				if ec := e.getConsole(int(ev.Fd)); ec != nil {
					ec.signalRead()
				}
			}
			// EPOLLOUT: write
			if ev.Events&(unix.EPOLLOUT|unix.EPOLLHUP|unix.EPOLLERR) != 0 {
				if ec := e.getConsole(int(ev.Fd)); ec != nil {
					ec.signalWrite()
				}
			}
		}
	}
}

func (e *Epoller) getConsole(ecfd int) *EpollConsole {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.fdMapping[ecfd]
}

// de-register from epoll
func (e *Epoller) CloseConsole(fd int) error {
	e.mu.Lock()
	delete(e.fdMapping, fd)
	e.mu.Unlock()
	return unix.EpollCtl(e.efd, unix.EPOLL_CTL_DEL, fd, &unix.EpollEvent{})
}

func (e *Epoller) Close() error {
	closeErr := os.ErrClosed
	e.closeOnce.Do(func() { closeErr = unix.Close(e.efd) })
	return closeErr
}

func (ec *EpollConsole) signalRead() {
	ec.readc.L.Lock()
	ec.readc.Signal()
	ec.readc.L.Unlock()
}

func (ec *EpollConsole) signalWrite() {
	ec.writec.L.Lock()
	ec.writec.Signal()
	ec.writec.L.Unlock()
}

func (ec *EpollConsole) Read(p []byte) (n int, err error) {
	var read int
	ec.readc.L.Lock()
	defer ec.readc.L.Unlock()
	for {
		read, err = ec.Console.Read(p[n:])
		n += read
		if err != nil {
			var hangup bool
			if perr, ok := err.(*os.PathError); ok {
				hangup = (perr.Err == unix.EAGAIN || perr.Err == unix.EIO)
			} else {
				hangup = (err == unix.EAGAIN || err == unix.EIO)
			}
			// if read side did not read anything yet, and epoll console is already closed, will break read loop
			if hangup && !(n == 0 && len(p) > 0 && ec.closed) {
				ec.readc.Wait()
				continue
			}
		}
		break
	}
	// if did not read anything
	if n == 0 && len(p) > 0 && err == nil {
		err = io.EOF
	}

	ec.readc.Signal() // singal read finsh
	return n, err
}

func (ec *EpollConsole) Write(p []byte) (n int, err error) {
	var written int
	ec.writec.L.Lock()
	defer ec.writec.L.Unlock()
	for {
		written, err = ec.Console.Write(p[n:])
		n += written
		if err != nil {
			var hangup bool
			if perr, ok := err.(*os.PathError); ok {
				hangup = (perr.Err == unix.EAGAIN || perr.Err == unix.EIO)
			} else {
				hangup = (err == unix.EAGAIN || err == unix.EIO)
			}
			if hangup {
				ec.writec.Wait()
				continue
			}
		}
		// break if not EAGAIN or IO error
		break
	}
	if n < len(p) && err != nil {
		err = io.ErrShortWrite
	}
	ec.writec.Signal()
	return n, err
}

// close func for closing ec's fd
func (ec *EpollConsole) Shutdown(close func(int) error) error {
	ec.readc.L.Lock()
	defer ec.readc.L.Unlock()
	ec.writec.L.Lock()
	defer ec.writec.L.Unlock()
	ec.readc.Broadcast()
	ec.writec.Broadcast()

	ec.closed = true
	return close(ec.sysfd)
}
