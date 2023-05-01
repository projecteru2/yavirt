package utils

import (
	"io/ioutil" //nolint
	"path/filepath"

	"github.com/projecteru2/yavirt/pkg/errors"
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
	var infos, err = ioutil.ReadDir(root)
	if err != nil {
		return errors.Trace(err)
	}

	for _, info := range infos {
		if err := fn(filepath.Join(root, info.Name()), info, nil); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
