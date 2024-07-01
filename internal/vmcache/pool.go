package vmcache

import (
	"sync/atomic"
	"time"

	"github.com/alphadose/haxmap"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket/dialers"
)

// the events libvirt emits when perfoming an vrish action are as follows:
//
// virsh start:    Resumed, Started
// virsh shutdown: Shutdown, Stopped
// virsh suspend:  Suspended
// virsh resume:   Resumed
const (
	DomainEventDefined = iota
	DomainEventUndefined
	DomainEventStarted
	DomainEventSuspended
	DomainEventResumed
	DomainEventStopped
	DomainEventShutdown
	DomainEventPMSuspended
)

func State2Str(state libvirt.DomainState) string {
	switch state {
	case libvirt.DomainNostate:
		return "NOSTATE"
	case libvirt.DomainRunning:
		return "RUNNING"
	case libvirt.DomainBlocked:
		return "BLOCKED"
	case libvirt.DomainPaused:
		return "PAUSED"
	case libvirt.DomainShutdown:
		return "SHUTDOWN"
	case libvirt.DomainShutoff:
		return "SHUTOFF"
	case libvirt.DomainCrashed:
		return "CRASHED"
	default:
		return "UNKNOWN"
	}
}

type LibvirtPool struct {
	index   atomic.Int64
	socket  string
	timeout time.Duration
	pool    *haxmap.Map[int64, *libvirt.Libvirt]
}

func (p *LibvirtPool) newLibvirt() (*libvirt.Libvirt, error) {
	opts := []dialers.LocalOption{
		dialers.WithSocket(p.socket),
		dialers.WithLocalTimeout(p.timeout),
	}
	dialer := dialers.NewLocal(opts...)
	l := libvirt.NewWithDialer(dialer)
	if err := l.Connect(); err != nil {
		return nil, err
	}
	idx := p.index.Add(1)
	p.pool.Set(idx, l)
	go func() {
		<-l.Disconnected()
		p.pool.Del(idx)
	}()
	return l, nil
}

func (p *LibvirtPool) Get() (*libvirt.Libvirt, error) {
	var ans *libvirt.Libvirt
	p.pool.ForEach(func(_ int64, v *libvirt.Libvirt) bool {
		ans = v
		return false
	})
	if ans == nil {
		return p.newLibvirt()
	}
	return ans, nil
}

func NewLibvirtPool(socket string, timeout time.Duration) *LibvirtPool {
	return &LibvirtPool{
		socket:  socket,
		timeout: timeout,
		pool:    haxmap.New[int64, *libvirt.Libvirt](),
	}
}
