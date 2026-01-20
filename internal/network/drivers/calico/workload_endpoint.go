package calico

import (
	"context"
	"net"
	"time"

	libcaliapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	libcalierr "github.com/projectcalico/calico/libcalico-go/lib/errors"
	libcalinames "github.com/projectcalico/calico/libcalico-go/lib/names"
	libcalinet "github.com/projectcalico/calico/libcalico-go/lib/net"
	libcaliopt "github.com/projectcalico/calico/libcalico-go/lib/options"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

// Get .
func (d *Driver) GetWEP(args types.EndpointArgs) (*libcaliapi.WorkloadEndpoint, error) {
	d.Lock()
	defer d.Unlock()
	return d.getWEP(args)
}

func (d *Driver) getWEP(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	var endpName string
	if endpName, err = d.generateEndpointName(args.Hostname, args.EndpointID); err != nil {
		return nil, errors.Wrap(err, "")
	}

	etcd.RetryTimedOut(func() error { //nolint
		cwe, err = d.WorkloadEndpoints().Get(context.Background(), args.Calico.Namespace, endpName, libcaliopt.GetOptions{})
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); ok { //nolint
				err = errors.Wrapf(terrors.ErrCalicoEndpointNotExists, "%s on %s", endpName, args.Hostname)
			}
		}

		return err
	}, 3)

	return
}

// Create .
func (d *Driver) CreateWEP(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	d.Lock()
	defer d.Unlock()

	return d.createWEP(args)
}

func (d *Driver) createWEP(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	if cwe, err = d.getCalicoWorkloadEndpoint(args); err != nil {
		return nil, errors.Wrap(err, "")
	}

	err = etcd.RetryTimedOut(func() error {
		var created, ce = d.WorkloadEndpoints().Create(context.Background(), cwe, libcaliopt.SetOptions{})
		if ce != nil {
			if _, ok := ce.(libcalierr.ErrorResourceAlreadyExists); !ok {
				return ce
			}
		}

		cwe = created

		return nil
	}, 3)

	return
}

// Update .
func (d *Driver) UpdateWEP(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	d.Lock()
	defer d.Unlock()
	return d.updateWEP(args)
}

func (d *Driver) updateWEP(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	cwe, err = d.getCalicoWorkloadEndpoint(args)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	cwe.UID = k8stypes.UID(args.Calico.UID)
	cwe.CreationTimestamp = k8smeta.NewTime(time.Now().UTC())

	err = etcd.RetryTimedOut(func() error {
		var updated, ue = d.WorkloadEndpoints().Update(context.Background(), cwe, libcaliopt.SetOptions{})
		if ue != nil {
			return ue
		}

		cwe = updated

		return nil
	}, 3)

	return
}

// Delete .
func (d *Driver) DeleteWEP(args types.EndpointArgs) error {
	d.Lock()
	defer d.Unlock()
	return d.deleteWEP(&args)
}

// func (we *Driver) delete(endpName, namespace string) (err error) {
func (d *Driver) deleteWEP(args *types.EndpointArgs) (err error) {
	endpName, err := d.generateEndpointName(args.Hostname, args.EndpointID)
	if err != nil {
		return errors.Wrap(err, "")
	}

	return etcd.RetryTimedOut(func() error {
		_, err := d.WorkloadEndpoints().Delete(
			context.TODO(),
			args.Calico.Namespace,
			endpName,
			libcaliopt.DeleteOptions{},
		)
		if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); !ok {
			return err
		}
		return nil
	}, 3)
}

func (d *Driver) ListWEP() ([]libcaliapi.WorkloadEndpoint, error) {
	var ident = libcalinames.WorkloadEndpointIdentifiers{
		Node: d.nodename,
	}
	namePrefix, _ := ident.CalculateWorkloadEndpointName(true)
	ans, err := d.WorkloadEndpoints().List(context.TODO(), libcaliopt.ListOptions{
		Name:   namePrefix,
		Prefix: true,
	})
	if err != nil {
		return nil, err
	}
	return ans.Items, nil
}

func (d *Driver) getCalicoWorkloadEndpoint(args types.EndpointArgs) (*libcaliapi.WorkloadEndpoint, error) {
	endpName, err := d.generateEndpointName(args.Hostname, args.EndpointID)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	ipNets, err := d.convCalicoIPNetworks(args.IPs)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	profile, err := d.getProfile(args.Calico.IPPool)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	wep := libcaliapi.NewWorkloadEndpoint()
	wep.Name = endpName
	wep.ObjectMeta.Namespace = args.Calico.Namespace
	wep.ObjectMeta.ResourceVersion = args.Calico.ResourceVersion
	wep.Spec.Endpoint = args.EndpointID
	wep.Spec.Node = args.Hostname
	wep.Spec.Orchestrator = OrchestratorID
	wep.Spec.Workload = OrchestratorID
	wep.Spec.InterfaceName = args.DevName
	wep.Spec.MAC = args.MAC
	wep.Spec.IPNetworks = append(wep.Spec.IPNetworks, ipNets...)
	wep.Spec.Profiles = d.mergeProfile(args.Calico.Profiles, profile)

	return wep, nil
}

func (d *Driver) mergeProfile(profiles []string, other string) []string {
	if len(profiles) < 1 {
		return []string{other}
	}

	for _, p := range profiles {
		if p == other {
			return profiles
		}
	}

	return append(profiles, other)
}

func (d *Driver) generateEndpointName(hostname, endpointID string) (string, error) {
	var ident = libcalinames.WorkloadEndpointIdentifiers{
		Node:         hostname,
		Orchestrator: OrchestratorID,
		Workload:     OrchestratorID,
		Endpoint:     endpointID,
	}
	return ident.CalculateWorkloadEndpointName(false)
}

func (d *Driver) convCalicoIPNetworks(ips []meta.IP) ([]string, error) {
	var ipNets = make([]string, len(ips))

	for i, ip := range ips {
		ipv4, _, err := netx.ParseCIDR(ip.CIDR())
		if err != nil {
			return nil, errors.Wrap(err, "")
		}

		ipNets[i] = libcalinet.IPNet{IPNet: net.IPNet{
			IP:   ipv4,
			Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8),
		}}.String()
	}

	return ipNets, nil
}

func (d *Driver) getProfile(poolName string) (string, error) {
	var pool, err = d.getIPPool(poolName)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return pool.ObjectMeta.Name, nil
}

// ConvIPs .
func ConvIPs(cwe *libcaliapi.WorkloadEndpoint) (ips []meta.IP, err error) {
	ips = make([]meta.IP, len(cwe.Spec.IPNetworks))

	for i, cidr := range cwe.Spec.IPNetworks {
		if ips[i], err = ParseCIDR(cidr); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}

	return ips, nil
}
