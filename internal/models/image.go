package models

import (
	"os"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Image wraps a few methods about Image.
type Image interface { //nolint
	GetName() string
	GetUser() string
	GetDistro() string
	GetID() string
	GetType() string
	GetHash() string
	UpdateHash() (string, error)

	NewSysVolume() *Volume
	Delete() error

	String() string
	Filepath() string
	Filename() string
}

// LoadImage loads an Image.
func LoadImage(name, user string) (Image, error) {
	if len(user) > 0 {
		return LoadUserImage(user, name)
	}
	return LoadSysImage(name)
}

// ListImages lists all images which belong to a specific user, or system-wide type.
func ListImages(user string) ([]Image, error) {
	if len(user) > 0 {
		return ListUserImages(user)
	}
	return ListSysImages()
}

// ImageExists whether the image file exists.
func ImageExists(img Image) (bool, error) {
	switch _, err := os.Stat(img.Filepath()); {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, errors.Trace(err)
	}
}
