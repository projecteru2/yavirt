package device

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

// NewAttrs .
func NewAttrs(name string, hwAddr net.HardwareAddr) netlink.LinkAttrs {
	var attrs = netlink.NewLinkAttrs()
	attrs.MTU = MTU
	attrs.TxQLen = Qlen
	attrs.Name = name
	attrs.HardwareAddr = hwAddr
	return attrs
}

func newHardwareAddr(linkType string) (net.HardwareAddr, error) {
	var mac string
	var err error

	switch linkType {
	case LinkTypeDummy:
		mac, err = newDummyMAC()

	case LinkTypeTuntap:
		fallthrough
	case LinkTypeTun:
		mac, err = newTuntapMAC()

	default:
		err = errors.Wrapf(terrors.ErrInvalidValue, "unexpected link type: %s", linkType)
	}

	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return net.ParseMAC(mac)
}

func newTuntapMAC() (string, error) {
	var buf, err = utils.RandBuf(3)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return fmt.Sprintf("fe:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2]), nil
}

func newDummyMAC() (string, error) {
	return utils.QemuMAC()
}
