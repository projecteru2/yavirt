package model

import "github.com/projecteru2/yavirt/pkg/meta"

// Networks .
type Networks []*Network

// RemoveNetwork .
func (ns *Networks) RemoveNetwork(mode, name string) Networks {
	removed := Networks{}

	for i := ns.Len() - 1; i >= 0; i-- {
		netw := (*ns)[i]
		if netw.Mode != mode || netw.Name != name {
			continue
		}

		removed = append(removed, netw)

		ns.Remove(i)
	}

	return removed
}

// Remove .
func (ns *Networks) Remove(index int) {
	switch {
	case ns.Len() < 1 || index >= ns.Len() || index < 0:
		return

	case index == 0:
		*ns = (*ns)[1:]

	default:
		*ns = append((*ns)[:index], (*ns)[index+1:]...)
	}
}

// Len .
func (ns Networks) Len() int { return len(ns) }

// Append .
func (ns *Networks) Append(ip meta.IP) {
	*ns = append(*ns, &Network{
		Mode: ip.NetworkMode(),
		Name: ip.NetworkName(),
		CIDR: ip.CIDR(),
		IP:   ip,
	})
}

// Network .
type Network struct {
	Mode string  `json:"mode"`
	Name string  `json:"name"`
	CIDR string  `json:"cidr"`
	IP   meta.IP `json:"-"`
}
