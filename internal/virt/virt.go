package virt

import (
	"os"
	"path/filepath"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// Cleanup cleans flocks up.
func Cleanup() error {
	files, err := os.ReadDir(configs.Conf.VirtFlockDir)
	if err != nil {
		return errors.Trace(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(configs.Conf.VirtFlockDir, f.Name())); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
