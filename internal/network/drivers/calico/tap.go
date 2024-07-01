package calico

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

func (d *Driver) createTap() (device.VirtLink, error) {
	var name, err = d.randTapName()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	for {
		var tap, err = d.dev.AddLink(device.LinkTypeTuntap, name)
		if err != nil {
			if errors.Is(err, terrors.ErrVirtLinkExists) {
				continue
			}

			return nil, errors.Wrap(err, "")
		}

		return tap, nil
	}
}

func (d *Driver) deleteTap(dev device.VirtLink) error {
	return d.dev.DeleteLink(dev.Name())
}

func (d *Driver) randTapName() (string, error) {
	var endpID, err = d.generateEndpointID()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	var name = fmt.Sprintf("%s%s", configs.Conf.Network.Calico.IFNamePrefix, endpID[:utils.Min(12, len(endpID))])

	return name, nil
}
