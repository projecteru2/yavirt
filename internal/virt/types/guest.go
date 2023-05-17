package types

import (
	pb "github.com/projecteru2/libyavirt/grpc/gen"
	virttypes "github.com/projecteru2/libyavirt/types"
)

type GuestCreateOption struct {
	CPU       int
	Mem       int64
	ImageName string
	ImageUser string
	Volumes   []virttypes.Volume
	DmiUUID   string
	Labels    map[string]string
	Cmd       []string
	Lambda    bool
	Stdin     bool
	Resources map[string][]byte
}

func ConvertGRPCCreateOptions(opts *pb.CreateGuestOptions) GuestCreateOption {
	ret := GuestCreateOption{
		CPU:       int(opts.Cpu),
		Mem:       opts.Memory,
		ImageName: opts.ImageName,
		ImageUser: opts.ImageUser,
		DmiUUID:   opts.DmiUuid,
		Labels:    opts.Labels,
		Cmd:       opts.Cmd,
		Lambda:    opts.Lambda,
		Stdin:     opts.Stdin,
		Resources: opts.Resources,
	}
	ret.Volumes = make([]virttypes.Volume, len(opts.Volumes))
	for i, vol := range opts.Volumes {
		ret.Volumes[i].Mount = vol.Mount
		ret.Volumes[i].Capacity = vol.Capacity
		ret.Volumes[i].IO = vol.Io
	}
	return ret
}
