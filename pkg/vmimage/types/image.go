package types

import (
	"fmt"
	"strings"

	"github.com/projecteru2/yavirt/pkg/vmimage/utils"
)

type PullPolicy string

const (
	PullPolicyAlways       = "Always"
	PullPolicyIfNotPresent = "IfNotPresent"
	PullPolicyNever        = "Never"
)

type OSInfo struct {
	Type    string `json:"type" default:"linux"`
	Distrib string `json:"distrib" default:"ubuntu"`
	Version string `json:"version"`
	Arch    string `json:"arch" default:"amd64"`
}

type Image struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Tag      string `json:"tag" description:"image tag, default:latest"`
	Private  bool   `json:"private"`
	OS       OSInfo `json:"os" description:"operating system info"`
	Size     int64  `json:"size"`
	Digest   string `json:"digest" description:"image digest"`
	Snapshot string `json:"snapshot" description:"image rbd snapshot"`

	ActualSize  int64
	VirtualSize int64
	LocalPath   string
}

func NewImage(fullname string) (*Image, error) {
	user, name, tag, err := utils.NormalizeImageName(fullname)
	if err != nil {
		return nil, err
	}
	return &Image{
		Username: user,
		Name:     name,
		Tag:      tag,
	}, nil
}

func (img *Image) Fullname() string {
	if img.Username == "" {
		return fmt.Sprintf("%s:%s", img.Name, img.Tag)
	} else { //nolint
		return fmt.Sprintf("%s/%s:%s", img.Username, img.Name, img.Tag)
	}
}

func (img *Image) RBDName() string {
	name := strings.ReplaceAll(img.Fullname(), "/", ".")
	return strings.ReplaceAll(name, ":", "-")
}

func (img *Image) Filepath() string {
	return img.LocalPath
}

func (img *Image) GetDigest() string {
	if img.Digest == "" {
		img.Digest, _ = utils.CalcDigestOfFile(img.LocalPath)
	}
	return img.Digest
}
