package calico

import (
	"sync"

	libcaliapi "github.com/projectcalico/libcalico-go/lib/apis/v3"

	calinet "github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/device"
)

// Handler .
type Handler struct {
	sync.Mutex

	cali *calinet.Driver
	dev  *device.Driver

	gateway                 *device.Dummy
	gatewayWorkloadEndpoint *libcaliapi.WorkloadEndpoint

	poolNames []string
	hostIP    string
}

// New .
func New(dev *device.Driver, driver *calinet.Driver, poolNames []string, hostIP string) *Handler {
	return &Handler{
		dev:       dev,
		cali:      driver,
		hostIP:    hostIP,
		poolNames: poolNames,
	}
}
