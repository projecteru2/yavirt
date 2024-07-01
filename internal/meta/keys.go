package meta

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/projecteru2/yavirt/configs"
)

const (
	hostPrefix     = "/hosts"
	guestPrefix    = "/guests"
	volPrefix      = "/vols"
	ipPrefix       = "/ips"
	imgPrefix      = "/imgs"
	uimgPrefix     = "/uimgs"
	snapshotPrefix = "/snapshots"
	ippPrefix      = "/ippools"
	ipblockPrefix  = "/blocks"
)

// HostCounterKey /<prefix>/hosts:counter
func HostCounterKey() string {
	return filepath.Join(configs.Conf.Etcd.Prefix, fmt.Sprintf("%s:counter", hostPrefix))
}

// HostGuestKey /<prefix>/hosts/<host name>/<guest ID>
func HostGuestKey(hostName, guestID string) string {
	return filepath.Join(HostGuestsPrefix(hostName), guestID)
}

// HostGuestsPrefix /<prefix>/hosts/<name>/
func HostGuestsPrefix(name string) string {
	return fmt.Sprintf("%s/", HostKey(name))
}

// HostKey /<prefix>/hosts/<name>
func HostKey(name string) string {
	return filepath.Join(configs.Conf.Etcd.Prefix, hostPrefix, name)
}

// GuestKey /<prefix>/guests/<id>
func GuestKey(id string) string {
	return filepath.Join(GuestsPrefix(), id)
}

// GuestsPrefix /<prefix>/guests/
func GuestsPrefix() string {
	return fmt.Sprintf("%s/", filepath.Join(configs.Conf.Etcd.Prefix, guestPrefix))
}

// VolumeKey /<prefix>/vols/<id>
func VolumeKey(id string) string {
	return filepath.Join(configs.Conf.Etcd.Prefix, volPrefix, id)
}

func SnapshotKey(id string) string {
	return filepath.Join(configs.Conf.Etcd.Prefix, snapshotPrefix, id)
}

// UserImageKey /<prefix>/uimgs/<user>/<name>
func UserImageKey(user, name string) string {
	return filepath.Join(UserImagePrefix(user), name)
}

// UserImagePrefix /<prefix/uimgs/<user>/
func UserImagePrefix(user string) string {
	return filepath.Join(configs.Conf.Etcd.Prefix, uimgPrefix, user)
}

// ImageKey /<prefix>/imgs/<name>
func SysImageKey(name string) string {
	return filepath.Join(SysImagePrefix(), name)
}

// SysImagePrefix /<prefix>/imgs/
func SysImagePrefix() string {
	return filepath.Join(configs.Conf.Etcd.Prefix, imgPrefix)
}

// OccupiedIPKey /<prefix>/ips/<subnet>/occupied/<ip>
func OccupiedIPKey(subnet, ip int64) string {
	var strIP = strconv.FormatInt(ip, 10)
	return filepath.Join(SubnetKey(subnet), "occupied", strIP)
}

// FreeIPKey /<prefix>/ips/<subnet>/free/<ip>
func FreeIPKey(subnet, ip int64) string {
	var strIP = strconv.FormatInt(ip, 10) //nolint:gomnd
	return filepath.Join(FreeIPPrefix(subnet), strIP)
}

// FreeIPPrefix /<prefix>/ips/<subnet>/free/
func FreeIPPrefix(subnet int64) string {
	return filepath.Join(SubnetKey(subnet), "free") + "/"
}

// IPALocKey /<prefix>/ips/<subnet>/lock
func IPALocKey(subnet int64) string {
	return filepath.Join(SubnetKey(subnet), "lock")
}

// SubnetKey /<prefix>/ips/<subnet>
func SubnetKey(subnet int64) string {
	var v = strconv.FormatInt(subnet, 10)
	return filepath.Join(configs.Conf.Etcd.Prefix, ipPrefix, v)
}

// IPBlockKey /<prefix>/ippool/<ipp name>/blocks/<subnet>
func IPBlockKey(ipp, subnet string) string {
	return filepath.Join(IPBlocksPrefix(ipp), subnet)
}

// IPBlocksPrefix /<prefix>/ippool/<ipp name>/blocks/
func IPBlocksPrefix(ipp string) string {
	return filepath.Join(IPPoolKey(ipp), ipblockPrefix)
}

// IPPoolLockKey /<prefix>/ippools/<name>/lock
func IPPoolLockKey(name string) string {
	return filepath.Join(IPPoolKey(name), "lock")
}

// IPPoolKey /<prefix>/ippools/<name>
func IPPoolKey(name string) string {
	return filepath.Join(IPPoolsPrefix(), name)
}

// IPPoolsPrefix /<prefix>/ippools/
func IPPoolsPrefix() string {
	return filepath.Join(configs.Conf.Etcd.Prefix, ippPrefix)
}
