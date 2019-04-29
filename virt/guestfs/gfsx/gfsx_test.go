package gfsx

import (
	"testing"

	"github.com/projecteru2/yavirt/test/assert"
)

func TestParseFstab(t *testing.T) {
	g := &Gfsx{}
	entries, err := g.parseFstab(`
  /dev/sda1 / ext4 defaults 0 1
 UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1
 # This's a comment line
LABEL=cloudimg-rootfs / ext4 defaults 0 1
`)
	assert.NilErr(t, err)
	assert.Equal(t, 3, len(entries))
	assert.Equal(t, "/dev/sda1 / ext4 defaults 0 1", entries["/dev/sda1"])
	assert.Equal(t, "UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1", entries["UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5"])
	assert.Equal(t, "LABEL=cloudimg-rootfs / ext4 defaults 0 1", entries["LABEL=cloudimg-rootfs"])
}
