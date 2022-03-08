package etcd

import "go.etcd.io/etcd/clientv3"

type delTxnEvent struct {
	*txnEvent
}

func newDelTxnEvent(keys []string, vers map[string]int64, opts ...clientv3.OpOption) *delTxnEvent {
	var ev = &delTxnEvent{txnEvent: newTxnEvent()}
	ev.vers = vers
	ev.data = map[string]string{}
	ev.opts = opts

	for _, k := range keys {
		ev.data[k] = ""
	}

	return ev
}

func (e *delTxnEvent) generate() ([]clientv3.Op, []clientv3.Cmp) {
	var limits = e.keysLimits()

	for k := range e.data {
		e.operations = append(e.operations, clientv3.OpDelete(k))

		if lmt, ok := limits[k]; ok {
			e.compares = append(e.compares, e.keyCmps(k, lmt)...)
		}
	}

	return e.operations, e.compares
}

type txnEvent struct {
	data       map[string]string
	vers       map[string]int64
	opts       []clientv3.OpOption
	txnErr     error
	operations []clientv3.Op
	compares   []clientv3.Cmp
}

func newTxnEvent() *txnEvent {
	return &txnEvent{
		operations: []clientv3.Op{},
		compares:   []clientv3.Cmp{},
	}
}

func (e *txnEvent) generate() ([]clientv3.Op, []clientv3.Cmp) {
	var limits = e.keysLimits()

	for k, v := range e.data {
		e.operations = append(e.operations, clientv3.OpPut(k, v, e.opts...))

		if lmt, ok := limits[k]; ok {
			e.compares = append(e.compares, e.keyCmps(k, lmt)...)
		}
	}

	return e.operations, e.compares
}

func (e *txnEvent) keyCmps(key string, limits keyLimits) []clientv3.Cmp {
	var cmps = []clientv3.Cmp{}
	for ver, expr := range limits {
		cmps = append(cmps, clientv3.Compare(clientv3.Version(key), expr, ver))
	}
	return cmps
}

func (e *txnEvent) keysLimits() map[string]keyLimits {
	var limits = map[string]keyLimits{}
	for k := range e.data {
		if ver, exists := e.vers[k]; exists {
			limits[k] = keyLimits{ver: "="}
		}
	}
	return limits
}

type keyLimits map[int64]string
