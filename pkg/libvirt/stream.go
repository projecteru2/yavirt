package libvirt

import (
	"io"

	"github.com/projecteru2/yavirt/pkg/utils"
)

type Stream struct {
	inStream  io.ReadWriteCloser
	outStream io.ReadWriteCloser
}

type ReadWrite struct {
	que *utils.BytesQueue
}

func (rw *ReadWrite) Write(p []byte) (n int, err error) {
	return rw.que.Write(p)
}

func (rw *ReadWrite) Close() error {
	rw.que.Close()
	return nil
}

func NewReadWrite() *ReadWrite {
	return &ReadWrite{
		que: utils.NewBytesQueue(),
	}
}

func (rw *ReadWrite) Read(p []byte) (n int, err error) {
	return rw.que.Read(p)
}

func NewStream() *Stream {
	return &Stream{
		inStream:  NewReadWrite(),
		outStream: NewReadWrite(),
	}
}

func (s *Stream) GetInReader() io.ReadWriteCloser {
	return s.inStream
}

func (s *Stream) GetOutWriter() io.ReadWriteCloser {
	return s.outStream
}

func (s *Stream) Recv(p []byte) (int, error) {
	n, err := s.outStream.Read(p)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Stream) Send(p []byte) (int, error) {
	n, err := s.inStream.Write(p)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Stream) Close() {
	if err := s.inStream.Close(); err != nil {
		return
	}
	if err := s.outStream.Close(); err != nil {
		return
	}
}
