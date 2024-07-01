package utils

import (
	"bytes"
	"container/list"
	"io"
	"sync"
)

type BytesQueue struct {
	cond *sync.Cond
	l    *list.List
	quit chan struct{}
}

func NewBytesQueue() *BytesQueue {
	return &BytesQueue{
		l:    list.New(),
		cond: sync.NewCond(&sync.Mutex{}),
		quit: make(chan struct{}),
	}
}

func (q *BytesQueue) Pop() ([]byte, error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	ele := q.l.Front()
	if ele == nil {
		select {
		case <-q.quit:
			return nil, io.EOF
		default:
			q.cond.Wait()
		}
	}
	ele = q.l.Front()
	if ele == nil {
		return nil, nil
	}
	q.l.Remove(ele)
	return ele.Value.([]byte), nil
}

func (q *BytesQueue) Read(p []byte) (total int, err error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	ele := q.l.Front()
	if ele == nil {
		select {
		case <-q.quit:
			return 0, io.EOF
		default:
			q.cond.Wait()
		}
	}
	ele = q.l.Front()
	if ele == nil {
		return 0, nil
	}

	bs := ele.Value.([]byte) //nolint
	total = copy(p, bs)
	if total < len(bs) {
		ele.Value = bs[total:]
	} else {
		q.l.Remove(ele)
	}
	return
}

func (q *BytesQueue) Write(p []byte) (int, error) {
	select {
	case <-q.quit:
		return 0, io.ErrClosedPipe
	default:
	}
	q.cond.L.Lock()
	q.l.PushBack(bytes.Clone(p))
	q.cond.L.Unlock()
	q.cond.Signal()
	return len(p), nil
}

func (q *BytesQueue) Close() {
	close(q.quit)
	q.cond.Broadcast()
}

func (q *BytesQueue) HandleHead(fn func([]byte) (int, error)) (total int, err error) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	ele := q.l.Front()
	if ele == nil {
		select {
		case <-q.quit:
			return 0, io.EOF
		default:
			return 0, nil
		}
	}
	ele = q.l.Front()
	if ele == nil {
		return 0, nil
	}

	bs := ele.Value.([]byte) //nolint
	total, err = fn(bs)
	if err != nil {
		return
	}
	if total < len(bs) {
		ele.Value = bs[total:]
	} else {
		q.l.Remove(ele)
	}
	return
}
