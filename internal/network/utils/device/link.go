package device

import (
	"strings"

	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/vishvananda/netlink"

	"github.com/cockroachdb/errors"
)

// VirtLink .
type VirtLink interface { //nolint
	Up() error
	Down() error

	Add() error

	BindAddr(*Addr) error
	ListAddr() (Addrs, error)

	AddRoute(dest, src string) error
	DeleteRoute(dest string) error
	ClearRoutes() error

	String() string
	MAC() string
	Name() string
	Index() int
}

func createVirtLink(linkType, name string, d *Driver) (VirtLink, error) {
	var hwAddr, err = newHardwareAddr(linkType)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var attrs = NewAttrs(name, hwAddr)

	switch linkType {
	case LinkTypeDummy:
		return createDummy(attrs, d), nil

	case LinkTypeTuntap:
		return createTuntap(attrs, d), nil

	default:
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "unexpected link type: %s", linkType)
	}
}

func newVirtLink(raw netlink.Link, d *Driver) (VirtLink, error) {
	switch raw.Type() {
	case LinkTypeDummy:
		return newDummy(raw, d), nil

	case LinkTypeTuntap:
		fallthrough
	case LinkTypeTun:
		return newTuntap(raw, d), nil

	default:
		return nil, errors.Wrapf(terrors.ErrInvalidValue, "unexpected link type: %s", raw.Type())
	}
}

// ShowLinkByIndex .
func (d *Driver) ShowLinkByIndex(index int) (VirtLink, error) {
	d.Lock()
	defer d.Unlock()

	var raw, err = d.LinkByIndex(index)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	return newVirtLink(raw, d)
}

// ShowLink .
func (d *Driver) ShowLink(name string) (VirtLink, error) {
	d.Lock()
	defer d.Unlock()
	return d.showLink(name)
}

func (d *Driver) showLink(name string) (VirtLink, error) {
	var raw, err = d.LinkByName(name)
	if err != nil {
		if err.Error() == "Link not found" {
			err = errors.Wrapf(terrors.ErrVirtLinkNotExists, "failed to get link %s", name)
		}

		return nil, errors.Wrap(err, name)
	}

	return newVirtLink(raw, d)
}

// ListLinks .
func (d *Driver) ListLinks() (VirtLinks, error) {
	d.Lock()
	defer d.Unlock()

	var raws, err = d.LinkList()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var links = make(VirtLinks, len(raws))

	for i, raw := range raws {
		if links[i], err = newVirtLink(raw, d); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	return links, nil
}

// AddLink .
func (d *Driver) AddLink(linkType, name string) (VirtLink, error) {
	var link, err = createVirtLink(linkType, name, d)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	if err := link.Add(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return link, nil
}

// delete link by name
func (d *Driver) DeleteLink(name string) error {
	var raw, err = d.LinkByName(name)
	if err != nil {
		if err.Error() == "Link not found" {
			err = errors.Wrapf(terrors.ErrVirtLinkNotExists, "failed to delete link %s", name)
		}
		return err
	}
	return d.LinkDel(raw)
}

// CheckLinkType .
func (d *Driver) CheckLinkType(linkType string) bool {
	return linkType == LinkTypeDummy ||
		linkType == LinkTypeTuntap ||
		linkType == LinkTypeTun
}

// VirtLinks .
type VirtLinks []VirtLink

// Len .
func (vls VirtLinks) Len() int {
	return len(vls)
}

func (vls VirtLinks) String() string {
	var strs = make([]string, vls.Len())

	for i, lnk := range vls {
		strs[i] = lnk.String()
	}

	return strings.Join(strs, "\n")
}
