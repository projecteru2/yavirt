package agent

import (
	"bytes"
	"context"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
)

// OpenFile .
func (a *Agent) OpenFile(ctx context.Context, path, mode string) (handle int, err error) {
	buf, err := a.qmp.OpenFile(ctx, path, mode)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}

	err = a.decode(buf, &handle)

	return
}

// CloseFile .
func (a *Agent) CloseFile(ctx context.Context, handle int) error {
	return a.qmp.CloseFile(ctx, handle)
}

// FlushFile .
func (a *Agent) FlushFile(ctx context.Context, handle int) error {
	return a.qmp.FlushFile(ctx, handle)
}

// ReadFile .
func (a *Agent) ReadFile(ctx context.Context, handle int, p []byte) (int, bool, error) {
	return a.qmp.ReadFile(ctx, handle, p)
}

// SeekFile .
func (a *Agent) SeekFile(ctx context.Context, handle int, offset int, whence int) (position int, eof bool, err error) {
	return a.qmp.SeekFile(ctx, handle, offset, whence)
}

// WriteFile .
func (a *Agent) WriteFile(ctx context.Context, handle int, buf []byte) error {
	return a.qmp.WriteFile(ctx, handle, buf)
}

// AppendLine .
func (a *Agent) AppendLine(ctx context.Context, filepath string, p []byte) error {
	file, err := OpenFile(ctx, a, filepath, "a")
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer file.Close(ctx)

	if _, err := file.WriteLine(ctx, p); err != nil {
		return errors.Wrap(err, "")
	}

	return file.Flush(ctx)
}

type File interface {
	Open(ctx context.Context) (err error)
	Flush(ctx context.Context) error
	Close(ctx context.Context) error
	Read(ctx context.Context, p []byte) (n int, err error)
	WriteLine(ctx context.Context, p []byte) (int, error)
	Write(ctx context.Context, p []byte) (n int, err error)
	Seek(ctx context.Context, offset, whence int) (pos int, err error)
	ReadAt(ctx context.Context, dest []byte, pos int) (n int, err error)
	Tail(ctx context.Context, n int) ([]byte, error)
	CopyTo(ctx context.Context, dst io.Writer) (int, error)
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
func OpenFile(ctx context.Context, agent *Agent, path, mode string) (File, error) {
	var wr = &file{
		agent: agent,
		path:  path,
		mode:  mode,
	}

	if err := wr.Open(ctx); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return wr, nil
}

// Open .
func (w *file) Open(ctx context.Context) (err error) {
	w.handle, err = w.agent.OpenFile(ctx, w.path, w.mode)
	return
}

// Flush .
func (w *file) Flush(ctx context.Context) error {
	return w.agent.FlushFile(ctx, w.handle)
}

// Close .
func (w *file) Close(ctx context.Context) error {
	return w.agent.CloseFile(ctx, w.handle)
}

// Read .
func (w *file) Read(ctx context.Context, p []byte) (n int, err error) {
	if w.eof {
		return 0, io.EOF
	}

	n, w.eof, err = w.agent.ReadFile(ctx, w.handle, p)

	return
}

// WriteLine .
func (w *file) WriteLine(ctx context.Context, p []byte) (int, error) {
	return w.Write(ctx, append(p, '\n'))
}

func (w *file) Write(ctx context.Context, p []byte) (n int, err error) {
	if len(p) < 1 {
		return
	}

	if err := w.agent.WriteFile(ctx, w.handle, p); err != nil {
		return 0, errors.Wrap(err, "")
	}

	return
}

// Seek .
func (w *file) Seek(ctx context.Context, offset, whence int) (pos int, err error) {
	pos, w.eof, err = w.agent.SeekFile(ctx, w.handle, offset, whence)
	return
}

// ReadAt .
func (w *file) ReadAt(ctx context.Context, dest []byte, pos int) (n int, err error) {
	_, err = w.Seek(ctx, pos, io.SeekStart)
	if err != nil {
		return
	}
	if w.eof {
		return 0, io.EOF
	}

	return w.Read(ctx, dest)
}

// Tail .
func (w *file) Tail(ctx context.Context, n int) ([]byte, error) {
	if n < 1 {
		return nil, errors.New("not valid tail")
	}

	size, err := w.Seek(ctx, 0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	if size == 1 {
		tmp := make([]byte, 1)
		_, err = w.Read(ctx, tmp)
		return tmp, err
	}

	tmp := [][]byte{}
	var lineStart int
	var buff bytes.Buffer
	lineEnd := size
	cursor := make([]byte, 1)
	for i := size - 2; i >= 0; i-- {
		if _, err = w.ReadAt(ctx, cursor, i); err != nil {
			return nil, err
		}

		if cursor[0] != '\n' && i != 0 {
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

		if _, err = w.Seek(ctx, lineStart, io.SeekStart); err != nil {
			return nil, err
		}

		newLine := make([]byte, lineEnd-lineStart)
		if _, err = w.Read(ctx, newLine); err != nil {
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

func (w *file) CopyTo(ctx context.Context, dst io.Writer) (int, error) {
	var total int
	for {
		buf := make([]byte, 65536)
		nRead, err := w.Read(ctx, buf)

		if err != nil && err != io.EOF {
			return total, errors.Wrap(err, "")
		}
		if nRead > 0 {
			if bytes.Contains(buf[:nRead], []byte("^]")) {
				log.WithFunc("CopyTo").Warnf(ctx, "[io.Scan] reader exited: %v", w)
				return total, errors.New("[CopyTo] reader got ^]")
			}
			nWrite, err := dst.Write(buf[:nRead])
			if err != nil {
				return total, errors.Wrap(err, "")
			}
			total += nWrite
		}
		if err == io.EOF {
			break
		}
	}
	return total, nil
}
