package model

import (
	"context"
	"fmt"
	"math"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/store"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// UserImage .
type UserImage struct {
	*SysImage
	User    string `json:"user"`
	Distro  string `json:"distro"`
	Version int64  `json:"version"`
}

// NewUserImage creates a new user captured image.
func NewUserImage(user, name string, size int64) *UserImage {
	img := &UserImage{SysImage: NewSysImage()}
	img.Name = name
	img.Size = size
	img.User = user
	return img
}

// ListUserImages list all images which belongs to the user.
func ListUserImages(user string) ([]Image, error) {
	ctx, cancel := meta.Context(context.TODO())
	defer cancel()

	prefix := meta.UserImagePrefix(user)
	data, vers, err := store.GetPrefix(ctx, prefix, math.MaxInt64)
	if err != nil {
		if errors.Contain(err, errors.ErrKeyNotExists) {
			return nil, nil
		}
		return nil, errors.Annotatef(err, "get sys images failed")
	}

	delete(data, prefix)

	return parseUserImages(data, vers)
}

func parseUserImages(data map[string][]byte, vers map[string]int64) ([]Image, error) {
	imgs := make([]Image, 0, len(data))

	for key, bytes := range data {
		ver, exists := vers[key]
		if !exists {
			return nil, errors.Annotatef(errors.ErrKeyBadVersion, key)
		}

		img := &UserImage{SysImage: NewSysImage()}
		if err := util.JSONDecode(bytes, img); err != nil {
			return nil, errors.Annotatef(err, "decode SysImage bytes %s failed", bytes)
		}

		img.SetVer(ver)

		imgs = append(imgs, img)
	}

	return imgs, nil
}

// LoadUserImage loads a user captured image.
func LoadUserImage(user, name string) (*UserImage, error) {
	i := NewUserImage(user, name, 0)
	return i, meta.Load(i)
}

// String .
func (i UserImage) String() string {
	return fmt.Sprintf("usr-image: %s, distro: %s, owner: %s", i.GetName(), i.GetDistro(), i.GetUser())
}

// GetType gets the image's type.
func (i UserImage) GetType() string {
	return ImageUser
}

// GetID gets the user captured image's ID which will be pushed to image hub.
func (i UserImage) GetID() string {
	return fmt.Sprintf("%s_%s", i.User, i.Name)
}

// GetUser gets the user captured image's owner name.
func (i UserImage) GetUser() string {
	return i.User
}

// Filepath gets a user captured image's absolute filepath.
func (i UserImage) Filepath() string {
	return i.JoinVirtPath(i.Filename())
}

// Filename generates a user captured image's filename without any path info.
func (i UserImage) Filename() string {
	return fmt.Sprintf("%s-%s-%s-%d.uimg", i.Distro, i.User, i.Name, i.Version)
}

// Delete removes the system-wide image
func (i UserImage) Delete() error {
	ctx, cancel := meta.Context(context.TODO())
	defer cancel()

	return store.Delete(
		ctx,
		[]string{i.MetaKey()},
		map[string]int64{i.MetaKey(): i.GetVer()},
	)
}

// NextVersion .
func (i *UserImage) NextVersion() error {
	// TODO
	// it should be distributed calculation/update, which means unique in global.
	return nil
}

// Save updates metadata.
func (i *UserImage) Save() error {
	return meta.Save(meta.Resources{i})
}

// Create creates a new user image to metadata.
func (i *UserImage) Create() error {
	i.Status = StatusRunning
	return meta.Create(meta.Resources{i})
}

// MetaKey .
func (i *UserImage) MetaKey() string {
	return meta.UserImageKey(i.User, i.Name)
}

// GetDistro gets the user captured image's distro.
func (i UserImage) GetDistro() string {
	return i.Distro
}
