package virt

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/internal/errors"
)

// Cleanup cleans flocks up.
func Cleanup() error {
	files, err := ioutil.ReadDir(config.Conf.VirtFlockDir)
	if err != nil {
		return errors.Trace(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(config.Conf.VirtFlockDir, f.Name())); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
