package calico

import (
	"net"

	libcaliapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// InitGateway .
func (h *Driver) InitGateway(gwName string) error {
	dev, err := device.New()
	if err != nil {
		return errors.Wrap(err, "")
	}

	h.Lock()
	defer h.Unlock()

	gw, err := dev.ShowLink(gwName)
	if err != nil {
		if terrors.IsVirtLinkNotExistsErr(err) {
			gw, err = dev.AddLink(device.LinkTypeDummy, gwName)
		}

		if err != nil {
			return errors.Wrap(err, "")
		}
	}

	var ok bool
	if h.gateway, ok = gw.(*device.Dummy); !ok {
		return errors.Wrapf(terrors.ErrInvalidValue, "expect *device.Dummy, but %v", gw)
	}

	if err := h.gateway.Up(); err != nil {
		return errors.Wrap(err, "")
	}

	if err := h.loadGateway(); err != nil {
		return errors.Wrap(err, "")
	}

	gwIPs, err := h.gatewayIPs()
	if err != nil {
		return errors.Wrap(err, "")
	}

	return h.bindGatewayIPs(gwIPs...)
}

// Gateway .
func (h *Driver) Gateway() *device.Dummy {
	h.Lock()
	defer h.Unlock()
	return h.gateway
}

// GatewayWorkloadEndpoint .
func (h *Driver) GatewayWorkloadEndpoint() *libcaliapi.WorkloadEndpoint {
	h.Lock()
	defer h.Unlock()
	return h.gatewayWorkloadEndpoint
}

func (h *Driver) bindGatewayIPs(ips ...meta.IP) error {
	for _, ip := range ips {
		var addr, err = h.dev.ParseCIDR(ip.CIDR())
		if err != nil {
			return errors.Wrap(err, "")
		}

		addr.IPNet = &net.IPNet{
			IP:   addr.IP,
			Mask: AllonesMask,
		}

		if err := h.gateway.BindAddr(addr); err != nil && !terrors.IsVirtLinkAddrExistsErr(err) {
			return errors.Wrap(err, "")
		}

		if err := h.gateway.ClearRoutes(); err != nil {
			return errors.Wrap(err, "")
		}
	}

	return nil
}

// RefreshGateway refreshes gateway data.
func (h *Driver) RefreshGateway() error {
	h.Lock()
	defer h.Unlock()
	return h.loadGateway()
}

func (h *Driver) loadGateway() error {
	hn := configs.Hostname()

	var args types.EndpointArgs
	args.Hostname = hn
	//TODO: better way to set namespace here
	args.Calico.Namespace = hn
	args.EndpointID = configs.Conf.Network.Calico.GatewayName

	wep, err := h.getWEP(args)
	if err != nil {
		if terrors.IsCalicoEndpointNotExistsErr(err) {
			return nil
		}
		return errors.Wrap(err, "")
	}

	h.gatewayWorkloadEndpoint = wep

	return nil
}

// GetGatewayIP gets a gateway IP which could serve the ip.
func (h *Driver) GetGatewayIP(ip meta.IP) (meta.IP, error) {
	h.Lock()
	defer h.Unlock()
	return h.getGatewayIP(ip)
}

func (h *Driver) getGatewayIP(ip meta.IP) (meta.IP, error) {
	if h.gatewayWorkloadEndpoint == nil {
		return nil, errors.Wrapf(terrors.ErrCalicoGatewayIPNotExists, "no such gateway WorkloadEndpoint")
	}

	for _, cidr := range h.gatewayWorkloadEndpoint.Spec.IPNetworks {
		var gwIP, err = ParseCIDR(cidr)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}

		if h.isUnderGateway(gwIP, ip) {
			return gwIP, nil
		}
	}

	return nil, errors.Wrapf(terrors.ErrCalicoGatewayIPNotExists, "for %s", ip)
}

func (h *Driver) isUnderGateway(gatewayIP, ip meta.IP) bool {
	var ipn = &net.IPNet{}
	ipn.IP = gatewayIP.NetIP()
	ipn.Mask = net.CIDRMask(ip.Prefix(), net.IPv4len*8)
	return ipn.Contains(ip.NetIP())
}

func (h *Driver) gatewayIPs() (ips []meta.IP, err error) {
	if h.gatewayWorkloadEndpoint != nil {
		ips, err = ConvIPs(h.gatewayWorkloadEndpoint)
	}
	return
}
