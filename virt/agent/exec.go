package agent

import (
	"context"
	"time"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/virt/agent/types"
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
	var st = <-a.exec(ctx, "echo", nil, true)
	return st.Error()
}

// ExecBatch .
func (a *Agent) ExecBatch(bat *config.Batch) error {
	var ctx = context.Background()
	var cancel context.CancelFunc
	if timeout := bat.Timeout.Duration(); timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var runOnce = bat.IsRunOnce()
	if runOnce {
		switch ran, err := a.isFile(ctx, bat.FlagFile); {
		case err != nil:
			return errors.Trace(err)
		case ran:
			return nil
		}
	}

	switch err := a.execBatch(ctx, bat); {
	case err != nil:
		return errors.Trace(err)
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

func (a *Agent) execBatch(ctx context.Context, bat *config.Batch) error {
	var cmds, err = bat.GetCommands()
	if err != nil {
		return errors.Trace(err)
	}

	for prog, args := range cmds {
		if err := a.execRetry(ctx, prog, args, bat); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (a *Agent) execRetry(ctx context.Context, prog string, args []string, bat *config.Batch) error {
	for {
		var st = <-a.exec(ctx, prog, args, true)

		var err = st.Error()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Annotatef(err, "run %s %s timeout", prog, args)
		default:
		}

		if !bat.Retry {
			return errors.Annotatef(err, "run %s %s error", prog, args)
		}

		log.ErrorStackf(err, "run %s %s error, retry it", prog, args)
		time.Sleep(bat.Interval.Duration())
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
	data, st.Err = a.qmp.Exec(prog, args, stdio)
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
				st.Err = errors.Annotatef(ctx.Err(), "exec %s error", prog)
				return

			case <-next.C:
				if st = a.execStatus(ret.Pid, stdio); st.Err != nil || st.Exited {
					return
				}
			}
		}
	}()

	return done
}

func (a *Agent) execStatus(pid int, _ bool) (st types.ExecStatus) {
	var data, err = a.qmp.ExecStatus(pid)
	if err != nil {
		st.Err = errors.Trace(err)
		return
	}

	if err := a.decode(data, &st); err != nil {
		st.Err = errors.Trace(err)
	}
	st.Pid = pid

	return
}
