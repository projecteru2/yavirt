package agent

import (
	"context"
	"encoding/json"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/virt/agent/types"
)

// Interface .
type Interface interface {
	Close() error
	Exec(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus
	ExecOutput(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus
	ExecBatch(bat *config.Batch) error
	Ping(ctx context.Context) error
	Touch(ctx context.Context, filepath string) error
	IsFile(ctx context.Context, filepath string) (bool, error)
	IsFolder(ctx context.Context, path string) (bool, error)
	RemoveAll(ctx context.Context, path string) error
	Grep(ctx context.Context, keyword, filepath string) (bool, error)
	OpenFile(path, mode string) (handle int, err error)
	CloseFile(handle int) error
	FlushFile(handle int) error
	ReadFile(handle int, p []byte) (int, bool, error)
	WriteFile(handle int, buf []byte) error
	SeekFile(handle int, offset int, whence int) (position int, eof bool, err error)
	AppendLine(filepath string, p []byte) error
	Blkid(ctx context.Context, dev string) (string, error)
	GetDiskfree(ctx context.Context, mnt string) (*types.Diskfree, error)
}

// Agent .
type Agent struct {
	qmp Qmp
}

// New .
func New(sockfile string) *Agent {
	return &Agent{
		qmp: newQmp(sockfile, true),
	}
}

// Close .
func (a *Agent) Close() (err error) {
	if a.qmp != nil {
		err = a.qmp.Close()
	}
	return
}

func (a *Agent) decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
