package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/pkg/utils"

	"github.com/projecteru2/yavirt/pkg/libvirt"
)

const maxBytesPerRead = 32 * utils.MB // ref https://www.qemu.org/docs/master/interop/qemu-ga-ref.html

// Qmp .
type Qmp interface { //nolint:interfacebloat
	Close() error

	Exec(ctx context.Context, cmd string, args []string, stdio bool) ([]byte, error)
	ExecStatus(ctx context.Context, pid int) ([]byte, error)

	OpenFile(ctx context.Context, path, mode string) ([]byte, error)
	FlushFile(ctx context.Context, handle int) error
	WriteFile(ctx context.Context, handle int, buf []byte) error
	ReadFile(ctx context.Context, handle int, p []byte) (read int, eof bool, err error)
	CloseFile(ctx context.Context, handle int) error
	SeekFile(ctx context.Context, handle int, offset int, whence int) (position int, eof bool, err error)
	FSFreezeAll(ctx context.Context) (nFS int, err error)
	FSFreezeList(ctx context.Context, mountpoints []string) (nFS int, err error)
	FSThawAll(ctx context.Context) (nFS int, err error)
	FSFreezeStatus(ctx context.Context) (status string, err error)
	GetName() string
}

type qmp struct {
	sync.Mutex

	// whether is guest-agent which means virsh qemu-agent-command,
	// the false value indicates virsh qemu-monitor-command.
	ga bool

	// sockfile string
	name string
	sock net.Conn
	virt libvirt.Libvirt
	dom  libvirt.Domain
}

type qmpResp struct {
	Event    *json.RawMessage
	Greeting *json.RawMessage
	Return   *json.RawMessage
	Error    *qmpError
}

type qmpError struct {
	Class string
	Desc  string
}

func (e *qmpError) Error() string {
	return fmt.Sprintf("QMP error %s: %s", e.Class, e.Desc)
}

func newQmp(name string, virt libvirt.Libvirt, ga bool) *qmp {
	return &qmp{
		name: name,
		virt: virt,
		ga:   ga,
	}
}

func (q *qmp) initIfNecessary() error {
	if q.dom != nil {
		return nil
	}
	dom, err := q.virt.LookupDomain(q.name)
	if err != nil {
		return err
	}
	q.dom = dom
	return nil
}

func (q *qmp) Exec(ctx context.Context, path string, args []string, output bool) ([]byte, error) {
	q.Lock()
	defer q.Unlock()

	var exArg = map[string]any{
		"path":           path,
		"capture-output": output,
	}
	if args != nil {
		exArg["arg"] = args
	}

	log.WithFunc("qmp.Exec").Debugf(ctx, "exec %s with %v", path, args)

	return q.exec(ctx, "guest-exec", exArg)
}

func (q *qmp) ExecStatus(ctx context.Context, pid int) ([]byte, error) {
	q.Lock()
	defer q.Unlock()
	return q.exec(ctx, "guest-exec-status", map[string]any{"pid": pid})
}

func (q *qmp) OpenFile(ctx context.Context, path, mode string) ([]byte, error) {
	q.Lock()
	defer q.Unlock()
	return q.exec(ctx, "guest-file-open", map[string]any{"path": path, "mode": mode})
}

func (q *qmp) CloseFile(ctx context.Context, handle int) (err error) {
	q.Lock()
	defer q.Unlock()
	_, err = q.exec(ctx, "guest-file-close", map[string]any{"handle": handle})
	return
}

func (q *qmp) FlushFile(ctx context.Context, handle int) (err error) {
	q.Lock()
	defer q.Unlock()
	_, err = q.exec(ctx, "guest-file-flush", map[string]any{"handle": handle})
	return
}

// ReadFile .
func (q *qmp) ReadFile(ctx context.Context, handle int, p []byte) (read int, eof bool, err error) {
	pcap := int64(cap(p))
	args := map[string]any{
		"handle": handle,
		"count":  utils.Min(maxBytesPerRead, pcap),
	}

	q.Lock()
	defer q.Unlock()

	for {
		var buf []byte
		if buf, err = q.exec(ctx, "guest-file-read", args); err != nil {
			return
		}

		resp := struct {
			Count int    `json:"count"`
			B64   string `json:"buf-b64"`
			EOF   bool   `json:"eof"`
		}{}
		if err = json.Unmarshal(buf, &resp); err != nil {
			return
		}

		eof = resp.EOF

		var src []byte
		if len(resp.B64) == 0 {
			return
		}

		if src, err = base64.StdEncoding.DecodeString(resp.B64); err != nil {
			return
		}

		read += copy(p[read:], src)

		if eof || int64(read) >= pcap {
			return
		}
	}
}

func (q *qmp) WriteFile(ctx context.Context, handle int, buf []byte) (err error) {
	q.Lock()
	defer q.Unlock()

	var b64 = base64.StdEncoding.EncodeToString(buf)
	_, err = q.exec(ctx, "guest-file-write", map[string]any{"handle": handle, "buf-b64": b64})

	return
}

func (q *qmp) FSFreezeAll(ctx context.Context) (nFS int, err error) {
	q.Lock()
	defer q.Unlock()

	var bs []byte
	if bs, err = q.exec(ctx, "guest-fsfreeze-freeze", nil); err != nil {
		return
	}
	nFS, err = strconv.Atoi(string(bs))

	return
}
func (q *qmp) FSFreezeList(ctx context.Context, mountpoints []string) (nFS int, err error) {
	q.Lock()
	defer q.Unlock()
	var args map[string]any
	if len(mountpoints) > 0 {
		args = map[string]any{"mountpoints": mountpoints}
	}
	var bs []byte
	if bs, err = q.exec(ctx, "guest-fsfreeze-freeze-list", args); err != nil {
		return
	}
	nFS, err = strconv.Atoi(string(bs))
	return
}

func (q *qmp) FSThawAll(ctx context.Context) (nFS int, err error) {
	q.Lock()
	defer q.Unlock()

	var bs []byte
	if bs, err = q.exec(ctx, "guest-fsfreeze-thaw", nil); err != nil {
		return
	}
	nFS, err = strconv.Atoi(string(bs))
	return
}

func (q *qmp) FSFreezeStatus(ctx context.Context) (status string, err error) {
	q.Lock()
	defer q.Unlock()

	var bs []byte
	if bs, err = q.exec(ctx, "guest-fsfreeze-status", nil); err != nil {
		return
	}
	status = string(bs[1 : len(bs)-1])
	return
}

func (q *qmp) exec(ctx context.Context, cmd string, args map[string]any) ([]byte, error) {
	if err := q.initIfNecessary(); err != nil {
		return nil, err
	}

	var buf, err = newQmpCmd(cmd, args).bytes()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	switch resp, err := q.req(ctx, buf); {
	case err != nil:
		return nil, errors.Wrap(err, "")

	case resp.Error != nil:
		return nil, errors.Wrapf(resp.Error, "failed to exec %s", cmd)

	default:
		return []byte(*resp.Return), nil
	}
}

// SeekFile .
func (q *qmp) SeekFile(ctx context.Context, handle int, offset int, whence int) (position int, eof bool, err error) {
	args := map[string]any{
		"handle": handle,
		"offset": offset,
		"whence": whence,
	}

	q.Lock()
	defer q.Unlock()

	var buf []byte
	if buf, err = q.exec(ctx, "guest-file-seek", args); err != nil {
		return
	}

	resp := struct {
		Position int  `json:"position"`
		EOF      bool `json:"eof"`
	}{}
	if err = json.Unmarshal(buf, &resp); err != nil {
		return
	}

	return resp.Position, resp.EOF, nil
}

func (q *qmp) Close() (err error) {
	if q.sock != nil {
		err = q.sock.Close()
	}
	return
}

func (q *qmp) req(ctx context.Context, cmd []byte) (qmpResp, error) {
	var resp qmpResp

	rs, err := q.dom.QemuAgentCommand(ctx, string(cmd))
	if err != nil {
		return resp, errors.Wrap(err, "")
	}

	if err := json.Unmarshal([]byte(rs), &resp); err != nil {
		return resp, errors.Wrap(err, "")
	}

	return resp, nil
}

func (q *qmp) GetName() string {
	return q.name
}

type qmpCmd struct {
	Name string         `json:"execute"`
	Args map[string]any `json:"arguments,omitempty"`
}

func newQmpCmd(name string, args map[string]any) (c qmpCmd) {
	c.Name = name
	c.Args = args
	return
}

func (c qmpCmd) bytes() ([]byte, error) {
	return json.Marshal(c)
}
