package model

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/util"
)

// SysImage indicates a system image
type SysImage struct {
	*Generic
	ParentName string `json:"parent,omitempty"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Hash       string `json:"sha256"`
}

// NewSysImage creates a new system-wide image.
func NewSysImage() *SysImage {
	return &SysImage{Generic: newGeneric()}
}

// ListSysImages lists all system-wide images.
func ListSysImages() ([]Image, error) {
	ctx, cancel := meta.Context(context.TODO())
	defer cancel()

	prefix := meta.SysImagePrefix()
	data, vers, err := store.GetPrefix(ctx, prefix, math.MaxInt64)
	if err != nil {
		if errors.Contain(err, errors.ErrKeyNotExists) {
			return nil, nil
		}
		return nil, errors.Annotatef(err, "get sys images failed")
	}

	delete(data, prefix)

	return parseSysImages(data, vers)
}

func parseSysImages(data map[string][]byte, vers map[string]int64) ([]Image, error) {
	imgs := make([]Image, 0, len(data))

	for key, bytes := range data {
		ver, exists := vers[key]
		if !exists {
			return nil, errors.Annotatef(errors.ErrKeyBadVersion, key)
		}

		img := NewSysImage()
		if err := util.JSONDecode(bytes, img); err != nil {
			return nil, errors.Annotatef(err, "decode SysImage bytes %s failed", bytes)
		}

		img.SetVer(ver)

		imgs = append(imgs, img)
	}

	return imgs, nil
}

// LoadSysImage loads a system-wide image.
func LoadSysImage(name string) (*SysImage, error) {
	img := NewSysImage()
	img.Name = name
	if err := meta.Load(img); err != nil {
		return nil, errors.Trace(err)
	}
	return img, nil
}

// String .
func (img *SysImage) String() string {
	return fmt.Sprintf("sys-image: %s", img.GetName())
}

// GetType gets the image's type.
func (img *SysImage) GetType() string {
	return ImageSys
}

// GetHash gets the image's hash.
func (img *SysImage) GetHash() string {
	return img.Hash
}

// UpdateHash update and return the image's hash .
func (img *SysImage) UpdateHash() (string, error) {
	exists, err := ImageExists(img)
	if err != nil {
		return "", err
	}
	if !exists {
		// TODO: Pull image?
		return "", errors.ErrImageFileNotExists
	}

	f, err := os.Open(img.Filepath())
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	img.Hash = fmt.Sprintf("%x", hash.Sum(nil))

	return img.Hash, img.Save()
}

// Save updated metadata.
func (img *SysImage) Save() error {
	return meta.Save(meta.Resources{img})
}

// GetID gets the image's ID which will be uploaded to image hub.
func (img *SysImage) GetID() string {
	return img.Name
}

// GetName gets image's name
func (img *SysImage) GetName() string {
	return img.Name
}

// GetUser gets the system-wide image's owner name
func (img *SysImage) GetUser() string {
	return ""
}

// Create .
func (img *SysImage) Create() error {
	img.Status = StatusRunning
	return meta.Create(meta.Resources{img})
}

// Delete removes the system-wide image.
func (img *SysImage) Delete() error {
	ctx, cancel := meta.Context(context.TODO())
	defer cancel()

	return store.Delete(
		ctx,
		[]string{img.MetaKey()},
		map[string]int64{img.MetaKey(): img.GetVer()},
	)
}

// MetaKey .
func (img *SysImage) MetaKey() string {
	return meta.SysImageKey(img.Name)
}

// NewSysVolume generates a new volume for OS' system disk.
func (img *SysImage) NewSysVolume() *Volume {
	return NewSysVolume(img.Size, img.Name)
}

// Filepath gets a system-wide image's absolute filepath.
func (img *SysImage) Filepath() string {
	return img.JoinVirtPath(img.Filename())
}

// Filename generates a system-wide image's filename without any path info.
func (img *SysImage) Filename() string {
	return fmt.Sprintf("%s.img", img.Name)
}

// GetDistro gets the system-wide image's distro.
func (img *SysImage) GetDistro() string {
	return img.Name[:6]
}
