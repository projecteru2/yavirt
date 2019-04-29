package grpcserver

import pb "github.com/projecteru2/libyavirt/grpc/gen"

// ExecuteGuestServerStream .
type ExecuteGuestServerStream struct {
	ID     string
	server pb.YavirtdRPC_AttachGuestServer
}

func (s *ExecuteGuestServerStream) Write(p []byte) (n int, err error) {
	msg := &pb.AttachGuestMessage{
		Id:   s.ID,
		Data: p,
	}
	return len(p), s.server.Send(msg)
}

func (s *ExecuteGuestServerStream) Read(p []byte) (n int, err error) {
	msg, err := s.server.Recv()
	if err != nil {
		return
	}
	copy(p, msg.ReplCmd)
	return len(msg.ReplCmd), nil
}

// Close .
func (s *ExecuteGuestServerStream) Close() error {
	return nil
}

// CatWriteCloser .
type CatWriteCloser struct {
	srv pb.YavirtdRPC_CatServer
}

// Write .
func (c *CatWriteCloser) Write(p []byte) (int, error) {
	return len(p), c.srv.Send(&pb.CatMessage{Data: p})
}

// Close .
func (c *CatWriteCloser) Close() error {
	return nil
}

// LogWriteCloser .
type LogWriteCloser struct {
	srv pb.YavirtdRPC_LogServer
}

// Write .
func (c *LogWriteCloser) Write(p []byte) (int, error) {
	return len(p), c.srv.Send(&pb.LogMessage{Data: p})
}

// Close .
func (c *LogWriteCloser) Close() error {
	return nil
}
