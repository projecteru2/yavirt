package calico

import (
	"net"

	libcaliapi "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/utils"
	calinet "github.com/projecteru2/yavirt/vnet/calico"
	"github.com/projecteru2/yavirt/vnet/device"
	"github.com/projecteru2/yavirt/vnet/types"
)

// InitGateway .
func (h *Handler) InitGateway(gwName string) error {
	dev, err := device.New()
	if err != nil {
		return errors.Trace(err)
	}

	h.Lock()
	defer h.Unlock()

	gw, err := dev.ShowLink(gwName)
	if err != nil {
		if errors.IsVirtLinkNotExistsErr(err) {
			gw, err = dev.AddLink(device.LinkTypeDummy, gwName)
		}

		if err != nil {
			return errors.Trace(err)
		}
	}

	var ok bool
	if h.gateway, ok = gw.(*device.Dummy); !ok {
		return errors.Annotatef(errors.ErrInvalidValue, "expect *device.Dummy, but %v", gw)
	}

	if err := h.gateway.Up(); err != nil {
		return errors.Trace(err)
	}

	if err := h.loadGateway(); err != nil {
		return errors.Trace(err)
	}

	gwIPs, err := h.gatewayIPs()
	if err != nil {
		return errors.Trace(err)
	}

	return h.bindGatewayIPs(gwIPs...)
}

// Gateway .
func (h *Handler) Gateway() *device.Dummy {
	h.Lock()
	defer h.Unlock()
	return h.gateway
}

// GatewayWorkloadEndpoint .
func (h *Handler) GatewayWorkloadEndpoint() *libcaliapi.WorkloadEndpoint {
	h.Lock()
	defer h.Unlock()
	return h.gatewayWorkloadEndpoint
}

func (h *Handler) bindGatewayIPs(ips ...meta.IP) error {
	for _, ip := range ips {
		var addr, err = h.dev.ParseCIDR(ip.CIDR())
		if err != nil {
			return errors.Trace(err)
		}

		addr.IPNet = &net.IPNet{
			IP:   addr.IPNet.IP,
			Mask: AllonesMask,
		}

		if err := h.gateway.BindAddr(addr); err != nil && !errors.IsVirtLinkAddrExistsErr(err) {
			return errors.Trace(err)
		}

		if err := h.gateway.ClearRoutes(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (h *Handler) addGatewayEndpoint(ip meta.IP) error {
	hn, err := util.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	var args types.EndpointArgs
	args.IPs = []meta.IP{ip}
	args.Device = h.gateway
	args.MAC = h.gateway.MAC()
	args.Hostname = hn
	args.EndpointID = config.Conf.CalicoGatewayName

	gwIPs, err := h.gatewayIPs()
	if err != nil {
		return errors.Trace(err)
	}

	var cwe *libcaliapi.WorkloadEndpoint

	if len(gwIPs) > 0 {
		args.IPs = append(args.IPs, gwIPs...)
		args.ResourceVersion = h.gatewayWorkloadEndpoint.ObjectMeta.ResourceVersion
		args.UID = string(h.gatewayWorkloadEndpoint.ObjectMeta.UID)
		args.Profiles = h.gatewayWorkloadEndpoint.Spec.Profiles
		cwe, err = h.cali.WorkloadEndpoint().Update(args)
	} else {
		cwe, err = h.cali.WorkloadEndpoint().Create(args)
	}

	if err != nil {
		return errors.Trace(err)
	}

	h.gatewayWorkloadEndpoint = cwe

	return nil
}

// RefreshGateway refreshes gateway data.
func (h *Handler) RefreshGateway() error {
	h.Lock()
	defer h.Unlock()
	return h.loadGateway()
}

func (h *Handler) loadGateway() error {
	hn, err := util.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	var args types.EndpointArgs
	args.Hostname = hn
	args.EndpointID = config.Conf.CalicoGatewayName

	wep, err := h.cali.WorkloadEndpoint().Get(args)
	if err != nil {
		if errors.IsCalicoEndpointNotExistsErr(err) {
			return nil
		}
		return errors.Trace(err)
	}

	h.gatewayWorkloadEndpoint = wep

	return nil
}

// GetGatewayIP gets a gateway IP which could serve the ip.
func (h *Handler) GetGatewayIP(ip meta.IP) (meta.IP, error) {
	h.Lock()
	defer h.Unlock()
	return h.getGatewayIP(ip)
}

func (h *Handler) getGatewayIP(ip meta.IP) (meta.IP, error) {
	if h.gatewayWorkloadEndpoint == nil {
		return nil, errors.Annotatef(errors.ErrCalicoGatewayIPNotExists, "no such gateway WorkloadEndpoint")
	}

	for _, cidr := range h.gatewayWorkloadEndpoint.Spec.IPNetworks {
		var gwIP, err = calinet.ParseCIDR(cidr)
		if err != nil {
			return nil, errors.Trace(err)
		}

		if h.isUnderGateway(gwIP, ip) {
			return gwIP, nil
		}
	}

	return nil, errors.Annotatef(errors.ErrCalicoGatewayIPNotExists, "for %s", ip)
}

func (h *Handler) isUnderGateway(gatewayIP, ip meta.IP) bool {
	var ipn = &net.IPNet{}
	ipn.IP = gatewayIP.NetIP()
	ipn.Mask = net.CIDRMask(ip.Prefix(), net.IPv4len*8) //nolint
	return ipn.Contains(ip.NetIP())
}

func (h *Handler) isCalicoGatewayIPNotExistsErr(err error) bool {
	return errors.Contain(err, errors.ErrCalicoGatewayIPNotExists)
}

func (h *Handler) gatewayIPs() (ips []meta.IP, err error) {
	if h.gatewayWorkloadEndpoint != nil {
		ips, err = calinet.ConvIPs(h.gatewayWorkloadEndpoint)
	}
	return
}
