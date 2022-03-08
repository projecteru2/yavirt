package types

import pb "github.com/projecteru2/libyavirt/grpc/gen"

type GuestCreateOption struct {
	CPU       int
	Mem       int64
	ImageName string
	ImageUser string
	Volumes   map[string]int64
	DmiUUID   string
	Labels    map[string]string
	Cmd       []string
	Lambda    bool
	Stdin     bool
}

func ConvertGRPCCreateOptions(opts *pb.CreateGuestOptions) GuestCreateOption {
	return GuestCreateOption{
		CPU:       int(opts.Cpu),
		Mem:       opts.Memory,
		ImageName: opts.ImageName,
		ImageUser: opts.ImageUser,
		Volumes:   opts.Volumes,
		DmiUUID:   opts.DmiUuid,
		Labels:    opts.Labels,
		Cmd:       opts.Cmd,
		Lambda:    opts.Lambda,
		Stdin:     opts.Stdin,
	}
}
