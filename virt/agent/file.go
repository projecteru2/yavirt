package agent

import (
	"bytes"
	"io"

	"github.com/juju/errors"
)

// OpenFile .
func (a *Agent) OpenFile(path, mode string) (handle int, err error) {
	buf, err := a.qmp.OpenFile(path, mode)
	if err != nil {
		return 0, errors.Trace(err)
	}

	err = a.decode(buf, &handle)

	return
}

// CloseFile .
func (a *Agent) CloseFile(handle int) error {
	return a.qmp.CloseFile(handle)
}

// FlushFile .
func (a *Agent) FlushFile(handle int) error {
	return a.qmp.FlushFile(handle)
}

// ReadFile .
func (a *Agent) ReadFile(handle int, p []byte) (int, bool, error) {
	return a.qmp.ReadFile(handle, p)
}

// SeekFile .
func (a *Agent) SeekFile(handle int, offset int, whence int) (position int, eof bool, err error) {
	return a.qmp.SeekFile(handle, offset, whence)
}

// WriteFile .
func (a *Agent) WriteFile(handle int, buf []byte) error {
	return a.qmp.WriteFile(handle, buf)
}

// AppendLine .
func (a *Agent) AppendLine(filepath string, p []byte) error {
	file, err := OpenFile(a, filepath, "a")
	if err != nil {
		return errors.Trace(err)
	}
	defer file.Close()

	if _, err := file.WriteLine(p); err != nil {
		return errors.Trace(err)
	}

	return file.Flush()
}

type File interface {
	Open() (err error)
	Flush() error
	Close() error
	Read(p []byte) (n int, err error)
	WriteLine(p []byte) (int, error)
	Write(p []byte) (n int, err error)
	Seek(offset, whence int) (pos int, err error)
	ReadAt(dest []byte, pos int) (n int, err error)
	Tail(n int) ([]byte, error)
}

// file .
type file struct {
	agent  *Agent
	path   string
	mode   string
	handle int
	eof    bool
}

// OpenFile .
func OpenFile(agent *Agent, path, mode string) (File, error) {
	var wr = &file{
		agent: agent,
		path:  path,
		mode:  mode,
	}

	if err := wr.Open(); err != nil {
		return nil, errors.Trace(err)
	}

	return wr, nil
}

// Open .
func (w *file) Open() (err error) {
	w.handle, err = w.agent.OpenFile(w.path, w.mode)
	return
}

// Flush .
func (w *file) Flush() error {
	return w.agent.FlushFile(w.handle)
}

// Close .
func (w *file) Close() error {
	return w.agent.CloseFile(w.handle)
}

// Read .
func (w *file) Read(p []byte) (n int, err error) {
	if w.eof {
		return 0, io.EOF
	}

	n, w.eof, err = w.agent.ReadFile(w.handle, p)

	return
}

// WriteLine .
func (w *file) WriteLine(p []byte) (int, error) {
	return w.Write(append(p, '\n'))
}

func (w *file) Write(p []byte) (n int, err error) {
	if len(p) < 1 {
		return
	}

	if err := w.agent.WriteFile(w.handle, p); err != nil {
		return 0, errors.Trace(err)
	}

	return
}

// Seek .
func (w *file) Seek(offset, whence int) (pos int, err error) {
	pos, w.eof, err = w.agent.SeekFile(w.handle, offset, whence)
	return
}

// ReadAt .
func (w *file) ReadAt(dest []byte, pos int) (n int, err error) {
	_, err = w.Seek(pos, io.SeekStart)
	if err != nil {
		return
	}
	if w.eof {
		return 0, io.EOF
	}

	return w.Read(dest)
}

// Tail .
func (w *file) Tail(n int) ([]byte, error) {
	if n < 1 {
		return nil, errors.New("not valid tail")
	}

	size, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	if size == 1 {
		tmp := make([]byte, 1)
		_, err = w.Read(tmp)
		return tmp, err
	}

	var tmp [][]byte
	var lineStart int
	var buff bytes.Buffer
	lineEnd := size
	cursor := make([]byte, 1)
	for i := size - 2; i >= 0; i-- { // nolint // start from the last second element
		if _, err = w.ReadAt(cursor, i); err != nil {
			return nil, err
		}

		if !(cursor[0] == '\n' || i == 0) {
			continue
		}

		if lineEnd == i {
			tmp = append(tmp, []byte{'\n'})
			continue
		}

		lineStart = i + 1
		if i == 0 {
			lineStart = 0
		}

		if _, err = w.Seek(lineStart, io.SeekStart); err != nil {
			return nil, err
		}

		newLine := make([]byte, lineEnd-lineStart)
		if _, err = w.Read(newLine); err != nil {
			return nil, err
		}
		tmp = append(tmp, newLine)

		if len(tmp) >= n {
			for j := len(tmp) - 1; j >= 0; j-- {
				buff.Write(tmp[j])
			}
			break
		}
		lineEnd = lineStart
	}

	return buff.Bytes(), nil
}
