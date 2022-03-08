package sh

import (
	"context"
	"io"
)

var shell Shell = shx{}

// GetShell .
func GetShell() Shell {
	return shell
}

// Shell .
type Shell interface {
	Copy(src, dest string) error
	Move(src, dest string) error
	Remove(fpth string) error
	Exec(ctx context.Context, name string, args ...string) error
	ExecInOut(ctx context.Context, env map[string]string, stdin io.Reader, name string, args ...string) ([]byte, []byte, error)
}

// Remove .
func Remove(fpth string) error {
	return shell.Remove(fpth)
}

// Move .
func Move(src, dest string) error {
	return shell.Move(src, dest)
}

// Copy .
func Copy(src, dest string) error {
	return shell.Copy(src, dest)
}

// ExecInOut .
func ExecInOut(ctx context.Context, env map[string]string, stdin io.Reader, name string, args ...string) ([]byte, []byte, error) {
	return shell.ExecInOut(ctx, env, stdin, name, args...)
}

// ExecContext .
func ExecContext(ctx context.Context, name string, args ...string) error {
	return shell.Exec(ctx, name, args...)
}
