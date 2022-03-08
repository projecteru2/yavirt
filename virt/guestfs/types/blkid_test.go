package types

import (
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestBlkidsExists(t *testing.T) {
	blkids := Blkids{}
	blkids.Add(&Blkid{Dev: "/dev/vda1", Label: "a", UUID: "34c4810a-3b02-11ec-a3d8-52540049d2ce"})
	blkids.Add(&Blkid{Dev: "/dev/vda2", Label: "b", UUID: "a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5"})
	blkids.Add(&Blkid{Dev: "/dev/vda3", Label: "c", UUID: "581f0760-3b02-11ec-a3d8-52540049d2ce"})

	assert.True(t, blkids.Exists("/dev/vda1"))
	assert.True(t, blkids.Exists("/dev/vda2"))
	assert.True(t, blkids.Exists("/dev/vda3"))
	assert.True(t, blkids.Exists("LABEL=a"))
	assert.True(t, blkids.Exists("LABEL=b"))
	assert.True(t, blkids.Exists("LABEL=c"))
	assert.True(t, blkids.Exists("UUID=34c4810a-3b02-11ec-a3d8-52540049d2ce"))
	assert.True(t, blkids.Exists("UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5"))
	assert.True(t, blkids.Exists("UUID=581f0760-3b02-11ec-a3d8-52540049d2ce"))

	assert.False(t, blkids.Exists("/dev/sda1"))
	assert.False(t, blkids.Exists("LABEL=d"))
	assert.False(t, blkids.Exists("UUID=6c8a6b06-3baa-11ec-b839-52540049d2ce"))
}
