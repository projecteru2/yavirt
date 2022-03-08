package calico

import (
	"fmt"

	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/vnet/device"
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

func (h *Handler) randTapName() (string, error) {
	var endpID, err = h.generateEndpointID()
	if err != nil {
		return "", errors.Trace(err)
	}

	var name = fmt.Sprintf("yap%s", endpID[:util.Min(12, len(endpID))]) //nolint

	return name, nil
}
