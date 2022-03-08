package calico

import (
	"fmt"
	"strings"

	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/internal/vnet/types"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// CreateEndpointNetwork .
func (h *Handler) CreateEndpointNetwork(args types.EndpointArgs) (types.EndpointArgs, func(), error) {
	h.Lock()
	defer h.Unlock()

	var err error

	if args.EndpointID, err = h.generateEndpointID(); err != nil {
		return args, nil, errors.Trace(err)
	}

	if args.Device, err = h.createTap(); err != nil {
		return args, nil, errors.Trace(err)
	}

	if err = args.Device.Up(); err != nil {
		return args, nil, errors.Trace(err)
	}

	if _, err = h.cali.WorkloadEndpoint().Create(args); err != nil {
		return args, nil, err
	}

	rollback := func() {
		if err := h.DeleteEndpointNetwork(args); err != nil {
			log.ErrorStackf(err, "delete endpoint %s failed", args.EndpointID)
		}
	}

	return args, rollback, err
}

// JoinEndpointNetwork .
func (h *Handler) JoinEndpointNetwork(args types.EndpointArgs) (func(), error) {
	if err := args.Check(); err != nil {
		return nil, errors.Trace(err)
	}

	h.Lock()
	defer h.Unlock()

	for _, ip := range args.IPs {
		ip.BindDevice(args.Device)

		if err := args.Device.AddRoute(ip.IPAddr(), h.hostIP); err != nil {
			if !errors.IsVirtLinkRouteExistsErr(err) {
				return nil, errors.Trace(err)
			}
		}
	}

	rollback := func() {
		dev := args.Device.Name()
		for _, ip := range args.IPs {
			cidr := fmt.Sprintf("%s/32", ip.IPAddr())
			if err := args.Device.DeleteRoute(cidr); err != nil {
				log.ErrorStackf(err, "delete route %s on %s failed", cidr, dev)
			}
		}
	}

	return rollback, nil
}

// DeleteEndpointNetwork .
func (h *Handler) DeleteEndpointNetwork(args types.EndpointArgs) error {
	h.Lock()
	defer h.Unlock()
	return h.cali.WorkloadEndpoint().Delete(args)
}

// GetEndpointDevice .
func (h *Handler) GetEndpointDevice(name string) (device.VirtLink, error) {
	dev, err := device.New()
	if err != nil {
		return nil, errors.Trace(err)
	}

	h.Lock()
	defer h.Unlock()

	return dev.ShowLink(name)
}

func (h *Handler) generateEndpointID() (string, error) {
	var uuid, err = util.UUIDStr()
	if err != nil {
		return "", errors.Trace(err)
	}
	return strings.ReplaceAll(uuid, "-", ""), nil
}
