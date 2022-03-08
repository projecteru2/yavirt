package calico

import (
	"context"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/meta"
	calinet "github.com/projecteru2/yavirt/vnet/calico"
	"github.com/projecteru2/yavirt/vnet/ipam"
)

// NewIP .
func (h *Handler) NewIP(name, cidr string) (meta.IP, error) {
	return calinet.ParseCIDR(cidr)
}

// AssignIP .
func (h *Handler) AssignIP() (ip meta.IP, err error) {
	h.Lock()
	defer h.Unlock()
	return h.assignIP(true)
}

func (h *Handler) assignIP(crossCalicoBlocks bool) (ip meta.IP, err error) {
	if ip, err = h.ipam().Assign(context.Background()); err != nil {
		return nil, errors.Trace(err)
	}

	var roll = ip
	defer func() {
		if err != nil && roll != nil {
			if re := h.releaseIPs(roll); re != nil {
				err = errors.Wrap(err, re)
			}
		}
	}()

	var gwIP meta.IP
	if gwIP, err = h.getGatewayIP(ip); err == nil {
		ip.BindGatewayIPNet(gwIP.IPNetwork())
		return ip, nil
	}

	switch {
	case !h.isCalicoGatewayIPNotExistsErr(err):
		return nil, errors.Annotatef(err, ip.CIDR())
	case !crossCalicoBlocks:
		return nil, errors.Annotatef(errors.ErrCalicoCannotCrossBlocks, ip.CIDR())
	}

	log.Warnf("%s doesn't belong any gateway, turn it into a gateway then", ip)
	if err = h.addGatewayEndpoint(ip); err != nil {
		return nil, errors.Trace(err)
	}

	// The assigned IP had truned into gateway workloadEndpoint,
	// so it shouldn't be rolled back even if there's a further error.
	roll = nil

	if err = h.bindGatewayIPs(ip); err != nil {
		log.Warnf("%s reserves as gateway addr, but bound device failed", ip)
		return nil, errors.Trace(err)
	}

	return h.assignIP(false)
}

// ReleaseIPs .
func (h *Handler) ReleaseIPs(ips ...meta.IP) error {
	h.Lock()
	defer h.Unlock()
	return h.releaseIPs(ips...)
}

func (h *Handler) releaseIPs(ips ...meta.IP) error {
	return h.ipam().Release(context.Background(), ips...)
}

// QueryIPs .
func (h *Handler) QueryIPs(ipns meta.IPNets) ([]meta.IP, error) {
	return h.ipam().Query(context.Background(), ipns)
}

func (h *Handler) ipam() ipam.Ipam {
	return h.cali.Ipam()
}

// QueryIPv4 .
func (h *Handler) QueryIPv4(ipv4 string) (meta.IP, error) {
	return nil, errors.Trace(errors.ErrNotImplemented)
}
