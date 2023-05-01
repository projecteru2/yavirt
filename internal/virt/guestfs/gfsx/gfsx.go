package gfsx

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/projecteru2/yavirt/internal/virt/guestfs"
	"github.com/projecteru2/yavirt/internal/virt/guestfs/types"
	"github.com/projecteru2/yavirt/pkg/errors"
	libguestfs "github.com/projecteru2/yavirt/third_party/guestfs"
)

// Gfsx .
type Gfsx struct {
	gfs    *libguestfs.Guestfs
	osDevs []string
}

// New .
func New(path string) (_ guestfs.Guestfs, err error) {
	gfsx := &Gfsx{osDevs: []string{}}
	if gfsx.gfs, err = libguestfs.Create(); err != nil {
		return
	}
	defer func() {
		if err != nil {
			gfsx.Close()
		}
	}()

	if err = gfsx.gfs.Add_drive(path, &libguestfs.OptargsAdd_drive{
		Readonly_is_set: false,
	}); err != nil {
		return
	}

	if err = gfsx.gfs.Launch(); err != nil {
		return
	}

	switch gfsx.osDevs, err = gfsx.gfs.Inspect_os(); {
	case err != nil:
		return
	case len(gfsx.osDevs) != 1:
		return nil, errors.Annotatef(errors.ErrInvalidValue, "%d OS in the image", len(gfsx.osDevs))
	}

	if err = gfsx.gfs.Mount(gfsx.osDevs[0], "/"); err != nil {
		return
	}

	return gfsx, nil
}

// Distro inspects the OS distro.
func (g *Gfsx) Distro() (string, error) {
	return g.gfs.Inspect_get_distro(g.osDevs[0])
}

// Close closes the driver.
func (g *Gfsx) Close() error {
	return g.gfs.Close()
}

// GetFstabEntries parses the /etc/fstab to build a map of the devices to the entries.
func (g *Gfsx) GetFstabEntries() (map[string]string, error) {
	cont, err := g.Cat(types.FstabFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return g.parseFstab(cont)
}

func (g *Gfsx) parseFstab(cont string) (map[string]string, error) {
	re, err := regexp.Compile(`^(.*?)\s`) //nolint
	if err != nil {
		return nil, errors.Trace(err)
	}

	entries := map[string]string{}

	lines := strings.Split(strings.TrimSpace(cont), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)

		// Skips the comment lines
		if strings.HasPrefix(l, "#") {
			continue
		}

		if ident := strings.TrimSpace(re.FindString(l)); len(ident) > 0 {
			entries[ident] = l
		}
	}

	return entries, nil
}

// GetBlkids populates all partitions' blkid.
func (g *Gfsx) GetBlkids() (types.Blkids, error) {
	fss, err := g.gfs.List_filesystems()
	if err != nil {
		return nil, errors.Trace(err)
	}

	blkids := types.Blkids{}
	for dev := range fss {
		blkid, err := g.Blkid(dev)
		if err != nil {
			return nil, errors.Annotatef(err, "get blkid %s failed", dev)
		}
		blkids.Add(blkid)
	}

	return blkids, nil
}

// Blkid gets the blkid of the dev.
func (g *Gfsx) Blkid(dev string) (*types.Blkid, error) {
	entries, err := g.gfs.Blkid(dev)
	if err != nil {
		return nil, errors.Trace(err)
	}

	blkid := &types.Blkid{Dev: dev}
	blkid.Label = entries[types.BlkLabel]
	blkid.UUID = entries[types.BlkUUID]
	blkid.Fstype = entries[types.BlkFstype]
	return blkid, nil
}

// Cat the path file as string.
func (g *Gfsx) Cat(path string) (string, error) {
	return g.gfs.Cat(path)
}

// Write writes data to the file
func (g *Gfsx) Write(path, data string) error {
	return g.gfs.Write(path, []byte(data))
}

// Remove deletes the file.
func (g *Gfsx) Remove(path string) error {
	return g.gfs.Rm_rf(path)
}

// Upload copies the file from the host to the image.
func (g *Gfsx) Upload(fileName, remoteFileName string) error {
	if err := g.gfs.Mkdir_p(filepath.Dir(remoteFileName)); err != nil {
		return err
	}
	return g.gfs.Upload(fileName, remoteFileName)
}

// IsDir checks the path whether is a directory.
func (g *Gfsx) IsDir(path string) (bool, error) {
	return g.gfs.Is_dir(path, nil)
}

// Tail .
func (g *Gfsx) Tail(n int, path string) ([]string, error) {
	return g.gfs.Tail_n(n, path)
}

// Read the path file.
func (g *Gfsx) Read(path string) ([]byte, error) {
	return g.gfs.Read_file(path)
}

// MakeDirectory .
func (g *Gfsx) MakeDirectory(path string, parent bool) error {
	if parent {
		return g.gfs.Mkdir_p(path)
	}
	return g.gfs.Mkdir(path)
}
