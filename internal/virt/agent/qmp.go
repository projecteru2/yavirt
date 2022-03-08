package agent

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const maxBytesPerRead = 32 * utils.MB // ref https://www.qemu.org/docs/master/interop/qemu-ga-ref.html

// Qmp .
type Qmp interface {
	Close() error

	Exec(cmd string, args []string, stdio bool) ([]byte, error)
	ExecStatus(pid int) ([]byte, error)

	OpenFile(path, mode string) ([]byte, error)
	FlushFile(handle int) error
	WriteFile(handle int, buf []byte) error
	ReadFile(handle int, p []byte) (read int, eof bool, err error)
	CloseFile(handle int) error
	SeekFile(handle int, offset int, whence int) (position int, eof bool, err error)
}

type qmp struct {
	sync.Mutex

	// whether is guest-agent which means virsh qemu-agent-command,
	// the false value indicates virsh qemu-monitor-command.
	ga bool

	sockfile string
	sock     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer

	greeting *json.RawMessage
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

func newQmp(sockfile string, ga bool) *qmp {
	return &qmp{
		sockfile: sockfile,
		ga:       ga,
	}
}

func (q *qmp) Exec(path string, args []string, output bool) ([]byte, error) {
	q.Lock()
	defer q.Unlock()

	var exArg = map[string]interface{}{
		"path":           path,
		"capture-output": output,
	}
	if args != nil {
		exArg["arg"] = args
	}

	log.Debugf("exec %s with %v", path, args)

	return q.exec("guest-exec", exArg)
}

func (q *qmp) ExecStatus(pid int) ([]byte, error) {
	q.Lock()
	defer q.Unlock()
	return q.exec("guest-exec-status", map[string]interface{}{"pid": pid})
}

func (q *qmp) OpenFile(path, mode string) ([]byte, error) {
	q.Lock()
	defer q.Unlock()
	return q.exec("guest-file-open", map[string]interface{}{"path": path, "mode": mode})
}

func (q *qmp) CloseFile(handle int) (err error) {
	q.Lock()
	defer q.Unlock()
	_, err = q.exec("guest-file-close", map[string]interface{}{"handle": handle})
	return
}

func (q *qmp) FlushFile(handle int) (err error) {
	q.Lock()
	defer q.Unlock()
	_, err = q.exec("guest-file-flush", map[string]interface{}{"handle": handle})
	return
}

// ReadFile .
func (q *qmp) ReadFile(handle int, p []byte) (read int, eof bool, err error) {
	pcap := int64(cap(p))
	args := map[string]interface{}{
		"handle": handle,
		"count":  utils.MinInt64(maxBytesPerRead, pcap),
	}

	q.Lock()
	defer q.Unlock()

	for {
		var buf []byte
		if buf, err = q.exec("guest-file-read", args); err != nil {
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

func (q *qmp) WriteFile(handle int, buf []byte) (err error) {
	q.Lock()
	defer q.Unlock()

	var b64 = base64.StdEncoding.EncodeToString(buf)
	_, err = q.exec("guest-file-write", map[string]interface{}{"handle": handle, "buf-b64": b64})

	return
}

func (q *qmp) exec(cmd string, args map[string]interface{}) ([]byte, error) {
	var buf, err = newQmpCmd(cmd, args).bytes()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := q.connect(); err != nil {
		return nil, errors.Trace(err)
	}

	switch resp, err := q.req(buf); {
	case err != nil:
		return nil, errors.Trace(err)

	case resp.Error != nil:
		return nil, errors.Trace(resp.Error)

	default:
		return []byte(*resp.Return), nil
	}
}

// SeekFile .
func (q *qmp) SeekFile(handle int, offset int, whence int) (position int, eof bool, err error) {
	args := map[string]interface{}{
		"handle": handle,
		"offset": offset,
		"whence": whence,
	}

	q.Lock()
	defer q.Unlock()

	var buf []byte
	if buf, err = q.exec("guest-file-seek", args); err != nil {
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

func (q *qmp) connect() error {
	if q.sock != nil {
		return nil
	}

	var sock, err = net.DialTimeout("unix", q.sockfile, configs.Conf.QMPConnectTimeout.Duration())
	if err != nil {
		return errors.Trace(err)
	}

	q.sock = sock
	q.reader = bufio.NewReader(q.sock)
	q.writer = bufio.NewWriter(q.sock)

	if !q.ga {
		if err := q.handshake(); err != nil {
			q.Close()
			return errors.Trace(err)
		}
	}

	return nil
}

func (q *qmp) handshake() error {
	return utils.Invoke([]func() error{
		q.greet,
		q.capabilities,
	})
}

func (q *qmp) capabilities() error {
	var cmd, err = newQmpCmd("qmp_capabilities", nil).bytes()
	if err != nil {
		return errors.Trace(err)
	}

	switch resp, err := q.req(cmd); {
	case err != nil:
		return errors.Trace(err)

	case resp.Return == nil:
		return errors.Errorf("QMP negotiation error")

	default:
		return nil
	}
}

func (q *qmp) greet() error {
	var buf, err = q.read()
	if err != nil {
		return errors.Trace(err)
	}

	var resp qmpResp

	switch err := json.Unmarshal(buf, &resp.Greeting); {
	case err != nil:
		return errors.Trace(err)
	case resp.Greeting == nil:
		return errors.Errorf("QMP greeting error")
	}

	q.greeting = resp.Greeting

	return nil
}

func (q *qmp) Close() (err error) {
	if q.sock != nil {
		err = q.sock.Close()
	}
	return
}

func (q *qmp) req(cmd []byte) (qmpResp, error) {
	var resp qmpResp

	if err := q.write(cmd); err != nil {
		return resp, errors.Trace(err)
	}

	var buf, err = q.read()
	if err != nil {
		return resp, errors.Trace(err)
	}

	if err := json.Unmarshal(buf, &resp); err != nil {
		return resp, errors.Trace(err)
	}

	return resp, nil
}

func (q *qmp) write(buf []byte) error {
	if _, err := q.writer.Write(append(buf, '\x0a')); err != nil {
		return errors.Trace(err)
	}

	if err := q.writer.Flush(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (q *qmp) read() ([]byte, error) {
	for {
		var buf, err = q.reader.ReadBytes('\n')
		if err != nil {
			return nil, errors.Trace(err)
		}

		var resp qmpResp
		if err := json.Unmarshal(buf, &resp); err != nil {
			return nil, errors.Trace(err)
		}

		if resp.Event != nil {
			log.Infof("recv event: %v", resp.Event)
			continue
		}

		return buf, nil
	}
}

type qmpCmd struct {
	Name string                 `json:"execute"`
	Args map[string]interface{} `json:"arguments,omitempty"`
}

func newQmpCmd(name string, args map[string]interface{}) (c qmpCmd) {
	c.Name = name
	c.Args = args
	return
}

func (c qmpCmd) bytes() ([]byte, error) {
	return json.Marshal(c)
}
