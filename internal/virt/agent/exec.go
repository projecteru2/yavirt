package agent

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/virt/agent/types"
)

// Grep .
func (a *Agent) Grep(ctx context.Context, keyword, filepath string) (bool, error) {
	var st = <-a.exec(ctx, "grep", []string{"-P", keyword, filepath}, true)
	return st.CheckReturnCode()
}

// IsFile .
func (a *Agent) IsFile(ctx context.Context, filepath string) (bool, error) {
	return a.isFile(ctx, filepath)
}

// IsFolder .
func (a *Agent) IsFolder(ctx context.Context, path string) (bool, error) {
	return a.isFolder(ctx, path)
}

// RemoveAll .
func (a *Agent) RemoveAll(ctx context.Context, path string) error {
	var st = <-a.exec(ctx, "rm", []string{"-rf", path}, true)
	return st.Error()
}

// MakeDirectory .
func (a *Agent) MakeDirectory(ctx context.Context, path string, parent bool) error {
	var paras []string
	if parent {
		paras = []string{"-p"}
	}
	var st = <-a.exec(ctx, "mkdir", append(paras, path), true)
	return st.Error()
}

// Touch .
func (a *Agent) Touch(ctx context.Context, filepath string) error {
	return a.touch(ctx, filepath)
}

func (a *Agent) touch(ctx context.Context, filepath string) error {
	var st = <-a.exec(ctx, "touch", []string{filepath}, true)
	return st.Error()
}

// Ping .
func (a *Agent) Ping(ctx context.Context) error {
	// linux and windows both have whoami.
	var st = <-a.exec(ctx, "whoami", nil, true)
	return st.Error()
}

// ExecBatch .
func (a *Agent) ExecBatch(bat *configs.Batch) error {
	var ctx = context.Background()
	var cancel context.CancelFunc
	if timeout := bat.Timeout; timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var runOnce = bat.IsRunOnce()
	if runOnce {
		switch ran, err := a.isFile(ctx, bat.FlagFile); {
		case err != nil:
			return errors.Wrap(err, "")
		case ran:
			return nil
		}
	}

	switch err := a.execBatch(ctx, bat); {
	case err != nil:
		return errors.Wrap(err, "")
	case runOnce:
		return a.touch(ctx, bat.FlagFile)
	default:
		return nil
	}
}

func (a *Agent) isFile(ctx context.Context, filepath string) (bool, error) {
	var st = <-a.exec(ctx, "test", []string{"-f", filepath}, true)
	return st.CheckReturnCode()
}

func (a *Agent) isFolder(ctx context.Context, path string) (bool, error) {
	var st = <-a.exec(ctx, "test", []string{"-d", path}, true)
	return st.CheckReturnCode()
}

func (a *Agent) execBatch(ctx context.Context, bat *configs.Batch) error {
	var cmds, err = bat.GetCommands()
	if err != nil {
		return errors.Wrap(err, "")
	}

	for prog, args := range cmds {
		if err := a.execRetry(ctx, prog, args, bat); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func (a *Agent) execRetry(ctx context.Context, prog string, args []string, bat *configs.Batch) error {
	for {
		var st = <-a.exec(ctx, prog, args, true)

		var err = st.Error()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Wrapf(err, "run %s %s timeout", prog, args)
		default:
		}

		if !bat.Retry {
			return errors.Wrapf(err, "run %s %s error", prog, args)
		}

		log.WithFunc("execRetry").Errorf(ctx, err, "run %s %s error, retry it", prog, args)
		time.Sleep(bat.Interval)
	}
}

// ExecOutput .
func (a *Agent) ExecOutput(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus {
	return a.exec(ctx, prog, args, true)
}

// Exec .
func (a *Agent) Exec(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus {
	return a.exec(ctx, prog, args, false)
}

func (a *Agent) exec(ctx context.Context, prog string, args []string, stdio bool) <-chan types.ExecStatus {
	var done = make(chan types.ExecStatus, 1)
	var st types.ExecStatus

	var data []byte
	data, st.Err = a.qmp.Exec(ctx, prog, args, stdio)
	if st.Err != nil {
		done <- st
		return done
	}

	var ret = struct {
		Pid int
	}{}
	if st.Err = a.decode(data, &ret); st.Err != nil {
		done <- st
		return done
	}

	st.Pid = ret.Pid

	go func() {
		defer func() {
			done <- st
		}()

		var next = time.NewTicker(time.Second)
		defer next.Stop()

		for i := 1; ; i++ {
			i %= 100

			select {
			case <-ctx.Done():
				st.Err = errors.Wrapf(ctx.Err(), "exec %s error", prog)
				return

			case <-next.C:
				if st = a.execStatus(ctx, ret.Pid, stdio); st.Err != nil || st.Exited {
					return
				}
			}
		}
	}()

	return done
}

func (a *Agent) execStatus(ctx context.Context, pid int, _ bool) (st types.ExecStatus) {
	var data, err = a.qmp.ExecStatus(ctx, pid)
	if err != nil {
		st.Err = errors.Wrap(err, "")
		return
	}

	if err := a.decode(data, &st); err != nil {
		st.Err = errors.Wrap(err, "")
	}
	st.Pid = pid

	return
}
