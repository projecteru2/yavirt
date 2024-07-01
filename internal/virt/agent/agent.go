package agent

import (
	"context"
	"encoding/json"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/virt/agent/types"
	"github.com/projecteru2/yavirt/pkg/libvirt"
)

// Interface .
type Interface interface { //nolint
	Close() error
	Exec(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus
	ExecOutput(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus
	ExecBatch(bat *configs.Batch) error
	Ping(ctx context.Context) error
	Touch(ctx context.Context, filepath string) error
	IsFile(ctx context.Context, filepath string) (bool, error)
	IsFolder(ctx context.Context, path string) (bool, error)
	RemoveAll(ctx context.Context, path string) error
	Grep(ctx context.Context, keyword, filepath string) (bool, error)
	OpenFile(ctx context.Context, path, mode string) (handle int, err error)
	CloseFile(ctx context.Context, handle int) error
	FlushFile(ctx context.Context, handle int) error
	ReadFile(ctx context.Context, handle int, p []byte) (int, bool, error)
	WriteFile(ctx context.Context, handle int, buf []byte) error
	SeekFile(ctx context.Context, handle int, offset int, whence int) (position int, eof bool, err error)
	AppendLine(ctx context.Context, filepath string, p []byte) error
	Blkid(ctx context.Context, dev string) (string, error)
	GetDiskfree(ctx context.Context, mnt string) (*types.Diskfree, error)
	FSFreezeAll(ctx context.Context) (int, error)
	FSThawAll(ctx context.Context) (int, error)
	FSFreezeStatus(ctx context.Context) (string, error)
}

// Agent .
type Agent struct {
	qmp Qmp
}

// New .
func New(name string, virt libvirt.Libvirt) *Agent {
	return &Agent{
		qmp: newQmp(name, virt, true),
	}
}

// Close .
func (a *Agent) Close() (err error) {
	if a.qmp != nil {
		err = a.qmp.Close()
	}
	return
}

func (a *Agent) decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
