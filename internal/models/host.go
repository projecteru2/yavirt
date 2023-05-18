package models

import (
	"fmt"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// Host .
// etcd keys:
//
//	/hosts:counter
//	/hosts/<host name>
type Host struct {
	*Generic

	ID          uint32 `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Subnet      int64  `json:"subnet"`
	CPU         int    `json:"cpu"`
	Memory      int64  `json:"mem"`
	Storage     int64  `json:"storage"`
	NetworkMode string `json:"network,omitempty"`
}

// LoadHost .
func LoadHost() (*Host, error) {
	host := &Host{
		Generic:     newGeneric(),
		Name:        configs.Conf.Host.Name,
		Type:        HostVirtType,
		Subnet:      int64(configs.Conf.Host.Subnet),
		CPU:         configs.Conf.Host.CPU,
		Memory:      int64(configs.Conf.Host.Memory),
		Storage:     int64(configs.Conf.Host.Storage),
		NetworkMode: configs.Conf.Host.NetworkMode,
	}
	dec, err := netx.IPv4ToInt(configs.Conf.Host.Addr)
	if err != nil {
		return nil, err
	}
	host.ID = uint32(dec)

	return host, nil
}

// NewHost .
func NewHost() *Host {
	return &Host{Generic: newGeneric()}
}

// MetaKey .
func (h *Host) MetaKey() string {
	return meta.HostKey(h.Name)
}

func (h *Host) String() string {
	return fmt.Sprintf("%d %s subnet: %d, cpu: %d, memory: %d, storage: %d",
		h.ID, h.Name, h.Subnet, h.CPU, h.Memory, h.Storage)
}

type hostGuest struct {
	*meta.Ver
	HostName string `json:"-"`
	GuestID  string `json:"-"`
}

func newHostGuest(hostName, guestID string) hostGuest {
	return hostGuest{
		Ver:      meta.NewVer(),
		HostName: hostName,
		GuestID:  guestID,
	}
}

func (hg hostGuest) MetaKey() string {
	return meta.HostGuestKey(hg.HostName, hg.GuestID)
}
