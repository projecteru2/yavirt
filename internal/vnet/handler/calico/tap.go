package calico

import (
	"fmt"

	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func (h *Handler) createTap() (device.VirtLink, error) {
	var name, err = h.randTapName()
	if err != nil {
		return nil, errors.Trace(err)
	}

	for {
		var tap, err = h.dev.AddLink(device.LinkTypeTuntap, name)
		if err != nil {
			if errors.Contain(err, errors.ErrVirtLinkExists) {
				continue
			}

			return nil, errors.Trace(err)
		}

		return tap, nil
	}
}

func (h *Handler) deleteTap(d device.VirtLink) error {
	return h.dev.DeleteLink(d.Name())
}

func (h *Handler) randTapName() (string, error) {
	var endpID, err = h.generateEndpointID()
	if err != nil {
		return "", errors.Trace(err)
	}

	var name = fmt.Sprintf("yap%s", endpID[:utils.Min(12, len(endpID))])

	return name, nil
}
