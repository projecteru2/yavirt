package virt

import (
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
)

// Cleanup cleans flocks up.
func Cleanup() error {
	files, err := os.ReadDir(configs.Conf.VirtFlockDir)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(configs.Conf.VirtFlockDir, f.Name())); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}
