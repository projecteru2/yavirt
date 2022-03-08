package model

import "github.com/projecteru2/yavirt/internal/virt/types"

type Manageable interface {
	GetAllGuests() ([]*Guest, error)
	GetNodeGuests(nodename string) ([]*Guest, error)
	LoadGuest(id string) (*Guest, error)
	CreateGuest(opts types.GuestCreateOption, host *Host, vols []*Volume) (*Guest, error)
	NewGuest(host *Host, img Image) (*Guest, error)
}

type Manager struct{}

var manager Manageable

func Setup() {
	manager = &Manager{}
}

func GetManager() Manageable {
	return manager
}

func SetManager(m Manageable) {
	manager = m
}
