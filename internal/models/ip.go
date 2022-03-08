package model

import "github.com/projecteru2/yavirt/internal/meta"

// IPs .
type IPs []meta.IP

func (ips *IPs) append(ip ...meta.IP) {
	*ips = append(*ips, ip...)
}

func (ips IPs) ipNets() meta.IPNets {
	return meta.ConvIPNets(ips)
}

func (ips IPs) setGuestID(id string) {
	for _, ip := range ips {
		ip.BindGuestID(id)
	}
}
