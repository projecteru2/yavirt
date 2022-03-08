package model

import (
	"context"
	"fmt"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/pkg/store"
)

// Host .
// etcd keys:
//     /hosts:counter
//     /hosts/<host name>
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
func LoadHost(hn string) (*Host, error) {
	var host = NewHost()
	host.Name = hn

	if err := meta.Load(host); err != nil {
		return nil, errors.Trace(err)
	}

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

// Create .
func (h *Host) Create() error {
	var id, err = genHostID()
	if err != nil {
		return errors.Trace(err)
	}

	h.ID = id
	h.Status = StatusRunning

	return meta.Create(meta.Resources{h})
}

func (h *Host) String() string {
	return fmt.Sprintf("%d %s subnet: %d, cpu: %d, memory: %d, storage: %d",
		h.ID, h.Name, h.Subnet, h.CPU, h.Memory, h.Storage)
}

func genHostID() (uint32, error) {
	var ctx, cancel = meta.Context(context.Background())
	defer cancel()

	var id, err = store.IncrUint32(ctx, meta.HostCounterKey())
	if err != nil {
		return 0, errors.Trace(err)
	}

	return id, nil
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
