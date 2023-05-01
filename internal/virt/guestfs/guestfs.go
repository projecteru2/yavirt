package guestfs

import "github.com/projecteru2/yavirt/internal/virt/guestfs/types"

// Guestfs .
type Guestfs interface { //nolint
	// Distro returns the OS distro name.
	Distro() (string, error)
	// Cat the file as string.
	Cat(string) (string, error)
	// Write writes the file.
	Write(string, string) error
	// Remove deletes the file.
	Remove(string) error
	// Close closes the driver.
	Close() error
	// Upload copies the file from the host to the image.
	Upload(fileName, remoteFileName string) error
	// IsDir checks the path whether is a directory.
	IsDir(path string) (bool, error)
	// GetFstabEntries parses the /etc/fstab to build a map of the devices to the entries.
	GetFstabEntries() (map[string]string, error)
	// GetBlkids populates all partitions' blkid.
	GetBlkids() (types.Blkids, error)
	// Tail n.
	Tail(n int, path string) ([]string, error)
	// Read .
	Read(path string) ([]byte, error)
	// MakeDirectory .
	MakeDirectory(path string, parent bool) error
}
