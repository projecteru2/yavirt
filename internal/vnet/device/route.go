package device

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// Route .
type Route struct {
	*netlink.Route

	*Driver
}

func newDefaultRoute(d *Driver, dest *net.IPNet) *Route {
	var raw = &netlink.Route{
		Table: RouteTableMain,
		Scope: RouteScopeLink,
		Dst:   dest,
	}
	return newRoute(raw, d)
}

func newRoute(raw *netlink.Route, d *Driver) *Route {
	return &Route{
		Route:  raw,
		Driver: d,
	}
}

func (r *Route) add() error {
	var err = r.RouteAdd(r.Route)

	if err != nil && isFileExistsErr(err) {
		return errors.Trace(errors.ErrVirtLinkRouteExists)
	}

	return err
}

func (r *Route) delete() error {
	return r.RouteDel(r.Route)
}

func (r *Route) String() string {
	var linkName, _ = r.linkName() //nolint

	if r.isGw() {
		return fmt.Sprintf("default via %s dev %s", r.Gw, linkName)
	}

	return fmt.Sprintf("%s dev %s proto %d scope %v src %s",
		r.Dst, linkName, r.Protocol, r.Scope, r.Src)
}

func (r *Route) isGw() bool {
	return len(r.Gw) > 0
}

func (r *Route) linkName() (string, error) {
	var name = "UNKNOWN"
	var link, err = r.Link()
	if err != nil {
		return name, errors.Trace(err)
	}
	return link.Name(), nil
}

// Link .
func (r *Route) Link() (VirtLink, error) {
	return r.ShowLinkByIndex(r.LinkIndex)
}

// Routes .
type Routes []*Route

func newRoutes(raw []netlink.Route, d *Driver) Routes {
	var routes = make(Routes, len(raw))
	for i, r := range raw {
		routes[i] = newRoute(&r, d) //nolint
	}
	return routes
}

func (r Routes) String() string {
	var strs = make([]string, r.Len())
	for i, x := range r {
		strs[i] = x.String()
	}
	return strings.Join(strs, ", ")
}

// Len .
func (r Routes) Len() int {
	return len(r)
}

// ListRoute .
func (d *Driver) ListRoute(dest string) (Routes, error) {
	d.Lock()
	defer d.Unlock()

	var ipn, err = netx.ParseCIDROrIP(dest)
	if err != nil {
		return nil, errors.Trace(err)
	}

	raw, err := d.RouteGet(ipn.IP)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return newRoutes(raw, d), nil
}

// ClearRoutes .
func (d *Driver) ClearRoutes(linkName string) error {
	var link, err = d.ShowLink(linkName)
	if err != nil {
		return errors.Annotatef(err, linkName)
	}
	return link.ClearRoutes()
}

// DeleteRoute .
func (d *Driver) DeleteRoute(dest string) error {
	var ipn, err = netx.ParseCIDROrIP(dest)
	if err != nil {
		return errors.Trace(err)
	}

	d.Lock()
	defer d.Unlock()

	return d.deleteRoute(ipn)
}

func (d *Driver) deleteRoute(dest *net.IPNet) error {
	var route = newDefaultRoute(d, dest)
	return route.delete()
}

// AddRoute .
func (d *Driver) AddRoute(dest, src, linkName string) error {
	d.Lock()
	defer d.Unlock()

	link, err := d.showLink(linkName)
	if err != nil {
		return errors.Trace(err)
	}

	return d.addRoute(dest, src, link)
}

func (d *Driver) addRoute(dest, src string, link VirtLink) error {
	ipn, err := netx.ParseCIDROrIP(dest)
	if err != nil {
		return errors.Trace(err)
	}

	var route = newDefaultRoute(d, ipn)
	route.LinkIndex = link.Index()

	if srcIP := net.ParseIP(src).To4(); srcIP.IsGlobalUnicast() {
		route.Src = srcIP
	}

	return route.add()
}
