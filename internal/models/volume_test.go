package models

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestDataVolume(t *testing.T) {
	fn := fmt.Sprintf("%s-id.vol", VolDataType)
	defaultFilepath := filepath.Join(configs.Conf.VirtDir, fn)
	mntFilepath := filepath.Join("/mnt", fn)

	cases := []struct {
		in, src, dst string
	}{
		{"/data", defaultFilepath, "/data"},
		{"data", defaultFilepath, "/data"},
		{"//data", defaultFilepath, "/data"},
		{" //data ", defaultFilepath, "/data"},

		{":/data", defaultFilepath, "/data"},
		{":data", defaultFilepath, "/data"},
		{" : data ", defaultFilepath, "/data"},

		{"/mnt:/data", mntFilepath, "/data"},
		{"mnt:/data", mntFilepath, "/data"},
		{"mnt:data", mntFilepath, "/data"},
		{"/mnt:data", mntFilepath, "/data"},
		{" /mnt : data ", mntFilepath, "/data"},
	}

	for _, c := range cases {
		vol, err := NewDataVolume(c.in, configs.Conf.MinVolumeCap)
		assert.NilErr(t, err)

		vol.ID = "id"
		assert.Equal(t, c.dst, vol.MountDir)
		assert.Equal(t, c.src, vol.Filepath())
	}
}
