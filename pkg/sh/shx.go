package sh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/cockroachdb/errors"
)

type shx struct{}

func (s shx) Remove(fpth string) error {
	return s.Exec(context.Background(), "rm", "-rf", fpth)
}

func (s shx) Move(src, dest string) error {
	return s.Exec(context.Background(), "mv", src, dest)
}

func (s shx) Copy(src, dest string) error {
	return s.Exec(context.Background(), "cp", src, dest)
}

func (s shx) ExecInOut(ctx context.Context, env map[string]string, stdin io.Reader, name string, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	err := cmd.Run()

	return stdout.Bytes(), stderr.Bytes(), err
}

func (s shx) Exec(ctx context.Context, name string, args ...string) error {
	var cmd = exec.CommandContext(ctx, name, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "")
	}

	slurp, err := io.ReadAll(stderr)
	if err != nil {
		return errors.Wrap(err, "")
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrap(err, string(slurp))
	}

	return nil
}
