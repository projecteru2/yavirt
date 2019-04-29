package util

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/log"
)

// ReadAll .
func ReadAll(fpth string) ([]byte, error) {
	f, err := os.Open(fpth)
	if err != nil {
		return nil, errors.Trace(err)
	}

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return buf, nil
}

// WriteTempFile .
func WriteTempFile(buf []byte) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), "temp-guest-*.xml")
	if err != nil {
		return "", errors.Trace(err)
	}

	if _, err := f.Write(buf); err != nil {
		return "", errors.Trace(err)
	}

	return f.Name(), nil
}

// Scan is tested to guarantee no goroutine leaking
func Scan(_ context.Context, reader io.Reader) <-chan []byte {
	ch := make(chan []byte)
	go func() {
		defer close(ch)

		for {
			p := make([]byte, 65536) //nolint // max(uint16) + 1
			n, err := reader.Read(p)
			if n > 0 {
				if bytes.Contains(p[:n], []byte("^]")) {
					log.Warnf("[io.Scan] reader exited: %v", reader)
					return
				}
				ch <- p[:n]
			}

			if err != nil {
				if err != io.EOF {
					log.Warnf("[io.Scan] error in reading %s: %s", reader, errors.Trace(err))
				}
				return
			}
		}

	}()

	return ch
}

// CopyIO is parallel to io.Copy execpt accepting context
func CopyIO(ctx context.Context, dst io.WriteCloser, src io.Reader) (written int, err error) {
	defer dst.Close()

	var n int
	ch := Scan(ctx, src)
	for {
		select {
		case <-ctx.Done():
			return
		case bytes, ok := <-ch:
			if !ok {
				return
			}
			if n, err = dst.Write(bytes); err != nil {
				log.Warnf("error in copy io: %s", errors.Trace(err))
				return
			}
			written += n
		}
	}
}
