package nic

import (
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/virt/agent"
)

// ConfigFile .
type ConfigFile interface {
	Save() error
	Close() error
}

// GenericConfigFile .
type GenericConfigFile struct {
	file agent.File
	path string
	ip   meta.IP
	dev  string
}

// Close .
func (w *GenericConfigFile) Close() (err error) {
	return w.file.Close()
}

func (w *GenericConfigFile) Write(buf []byte) (err error) {
	defer func() {
		if err == nil {
			err = w.file.Flush()
		}
	}()

	_, err = w.file.Write(buf)

	return
}
