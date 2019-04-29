package types

import "fmt"

// Blkid indicates a blkid of guestfs
type Blkid struct {
	Dev    string
	Label  string
	UUID   string
	Fstype string
}

// Blkids indicates a group of blkid.
type Blkids map[string]*Blkid

// Add adds a Blkid to the Blkids.
func (b Blkids) Add(blkid *Blkid) {
	b[blkid.Dev] = blkid
	b[fmt.Sprintf("LABEL=%s", blkid.Label)] = blkid
	b[fmt.Sprintf("UUID=%s", blkid.UUID)] = blkid
}

// Exists checks the dev whether is existed in the blkids.
func (b Blkids) Exists(ident string) bool {
	_, exists := b[ident]
	return exists
}
