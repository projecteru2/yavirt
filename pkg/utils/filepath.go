package utils

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
)

// BaseFilename .
func BaseFilename(fpth string) (fn string, ext string) {
	var base = filepath.Base(fpth)
	return PartRight(base, ".")
}

// AbsDir .
func AbsDir(fpth string) (string, error) {
	if filepath.IsAbs(fpth) {
		return fpth, nil
	}
	return filepath.Abs(fpth)
}

// Walk .
// Re-implements as filepath.Walk doesn't follow symlinks.
func Walk(root string, fn filepath.WalkFunc) error {
	var entries, err = os.ReadDir(root)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return errors.Wrap(err, "")
		}
		if err := fn(filepath.Join(root, entry.Name()), info, nil); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
