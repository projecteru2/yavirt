package calico

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	calicoMTU = 1500
)

// CreateEndpointNetwork .
func (h *Driver) CreateEndpointNetwork(args types.EndpointArgs) (types.EndpointArgs, func() error, error) {
	// Create network policy if necessary
	// TODO: maybe we can create network policy when create new user.
	if err := h.CreateNetworkPolicy(args.Calico.Namespace); err != nil {
		return args, nil, errors.Wrapf(err, "failed to create network policy")
	}

	h.Lock()
	defer h.Unlock()

	// alloc an ip for this endpoint
	ip, err := h.assignIP(&args)
	if err != nil {
		return args, nil, errors.Wrap(err, "")
	}
	rollbackIP := func() error {
		return h.releaseIPs(ip)
	}

	args.IPs = append(args.IPs, ip)

	if args.EndpointID, err = h.generateEndpointID(); err != nil {
		return args, rollbackIP, errors.Wrap(err, "")
	}

	dev, err := h.createTap()
	if err != nil {
		return args, rollbackIP, errors.Wrap(err, "")
	}
	args.DevName = dev.Name()
	if err = dev.Up(); err != nil {
		return args, rollbackIP, errors.Wrap(err, "")
	}

	// qemu will create TAP device when start, so we can delete it here
	defer func() {
		// try to delete tap device, we ignore the error
		err = h.deleteTap(dev)
		log.Debugf(context.TODO(), "After delete tap device(%v): %v", dev.Name(), err)
	}()

	if _, err = h.createWEP(args); err != nil {
		return args, rollbackIP, err
	}
	rollback := func() error {
		err1 := rollbackIP()
		err2 := h.DeleteEndpointNetwork(args)
		return errors.CombineErrors(err1, err2)
	}
	args.MTU = calicoMTU
	return args, rollback, err
}

// JoinEndpointNetwork .
func (h *Driver) JoinEndpointNetwork(args types.EndpointArgs) (func() error, error) {
	if err := args.Check(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	h.Lock()
	defer h.Unlock()

	devDriver, err := device.New()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	dev, err := devDriver.ShowLink(args.DevName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get link")
	}

	for _, ip := range args.IPs {
		ip.BindDevice(dev)

		if err := dev.AddRoute(ip.IPAddr(), h.hostIP); err != nil {
			if !terrors.IsVirtLinkRouteExistsErr(err) {
				return nil, errors.Wrap(err, "")
			}
		}
	}

	if h.dhcp != nil && len(args.IPs) > 0 {
		ip := args.IPs[0]
		if err := h.dhcp.AddInterface(dev.Name(), ip.NetIP(), ip.IPNetwork()); err != nil {
			return nil, errors.Wrap(err, "Failed to add interface to dhcp server")
		}
	}
	rollback := func() error {
		var err error
		for _, ip := range args.IPs {
			cidr := fmt.Sprintf("%s/32", ip.IPAddr())
			if err1 := dev.DeleteRoute(cidr); err1 != nil {
				err = errors.CombineErrors(err, err1)
			}
		}
		return err
	}

	return rollback, nil
}

// DeleteEndpointNetwork .
func (h *Driver) DeleteEndpointNetwork(args types.EndpointArgs) error {
	h.Lock()
	defer h.Unlock()
	err1 := h.releaseIPs(args.IPs...)
	err2 := h.deleteWEP(&args)
	return errors.CombineErrors(err1, err2)
}

func (h *Driver) generateEndpointID() (string, error) {
	var uuid, err = utils.UUIDStr()
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return strings.ReplaceAll(uuid, "-", ""), nil
}
