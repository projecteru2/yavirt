package vmcache

import (
	"testing"

	"github.com/digitalocean/go-libvirt"
	"github.com/stretchr/testify/assert"
	"libvirt.org/go/libvirtxml"
)

func TestDomainCacheEntryCopy(t *testing.T) {
	original := &DomainCacheEntry{
		Name:     "example",
		UUID:     "123456",
		State:    libvirt.DomainRunning,
		VNCPort:  5900,
		GPUAddrs: nil,
		Schema:   &libvirtxml.Domain{},
		AppName:  "myApp",
		IP:       "192.168.1.1",
		EruName:  "ERU",
		UserName: "user",
		UserID:   "123",
	}
	original.GPUAddrs = make([]string, 0, 10)
	original.GPUAddrs = append(original.GPUAddrs, "GPU1", "GPU2")

	// Make a copy of the original
	copied := original.Copy()

	// Use assert to check if the copy is equal to the original
	assert.Equal(t, original, copied, "Copied object should be equal to the original")

	// Modify the original
	original.VNCPort = 5901

	// Use assert to verify that the copied object is not affected by the modification
	assert.NotEqual(t, original, copied, "Copied object should not be affected by modifications to the original")
	assert.Equal(t, copied.VNCPort, 5900)

	original.GPUAddrs = append(original.GPUAddrs, "GPU3")
	assert.Equal(t, copied.GPUAddrs, []string{"GPU1", "GPU2"})
	assert.Equal(t, original.GPUAddrs, []string{"GPU1", "GPU2", "GPU3"})
}
