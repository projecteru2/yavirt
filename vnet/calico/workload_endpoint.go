package calico

import (
	"context"
	"net"
	"time"

	libcaliapi "github.com/projectcalico/libcalico-go/lib/apis/v3"
	libcalierr "github.com/projectcalico/libcalico-go/lib/errors"
	libcalinames "github.com/projectcalico/libcalico-go/lib/names"
	libcalinet "github.com/projectcalico/libcalico-go/lib/net"
	libcaliopt "github.com/projectcalico/libcalico-go/lib/options"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/projecteru2/yavirt/internal/errors"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/netx"
	"github.com/projecteru2/yavirt/store/etcd"
	"github.com/projecteru2/yavirt/vnet/types"
)

// WorkloadEndpoint .
type WorkloadEndpoint struct {
	*Driver
}

func newWorkloadEndpoint(driver *Driver) *WorkloadEndpoint {
	return &WorkloadEndpoint{Driver: driver}
}

// Get .
func (we *WorkloadEndpoint) Get(args types.EndpointArgs) (*libcaliapi.WorkloadEndpoint, error) {
	we.Lock()
	defer we.Unlock()
	return we.get(args)
}

// Create .
func (we *WorkloadEndpoint) Create(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	we.Lock()
	defer we.Unlock()

	if cwe, err = we.getCalicoWorkloadEndpoint(args); err != nil {
		return nil, errors.Trace(err)
	}

	err = etcd.RetryTimedOut(func() error {
		var created, ce = we.WorkloadEndpoints().Create(context.Background(), cwe, libcaliopt.SetOptions{})
		if ce != nil {
			if _, ok := ce.(libcalierr.ErrorResourceAlreadyExists); !ok {
				return ce
			}
		}

		cwe = created

		return nil
	}, 3) //nolint

	return
}

func (we *WorkloadEndpoint) get(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	var endpName string
	if endpName, err = we.generateEndpointName(args.Hostname, args.EndpointID); err != nil {
		return nil, errors.Trace(err)
	}

	etcd.RetryTimedOut(func() error { //nolint
		cwe, err = we.WorkloadEndpoints().Get(context.Background(), args.Hostname, endpName, libcaliopt.GetOptions{})
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); ok { //nolint
				err = errors.Annotatef(errors.ErrCalicoEndpointNotExists, "%s on %s", endpName, args.Hostname)
			}
		}

		return err
	}, 3) //nolint

	return
}

// Update .
func (we *WorkloadEndpoint) Update(args types.EndpointArgs) (cwe *libcaliapi.WorkloadEndpoint, err error) {
	we.Lock()
	defer we.Unlock()

	cwe, err = we.getCalicoWorkloadEndpoint(args)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cwe.ObjectMeta.UID = k8stypes.UID(args.UID)
	cwe.ObjectMeta.CreationTimestamp = k8smeta.NewTime(time.Now().UTC())

	err = etcd.RetryTimedOut(func() error {
		var updated, ue = we.WorkloadEndpoints().Update(context.Background(), cwe, libcaliopt.SetOptions{})
		if ue != nil {
			return ue
		}

		cwe = updated

		return nil
	}, 3) //nolint

	return
}

// Delete .
func (we *WorkloadEndpoint) Delete(args types.EndpointArgs) error {
	var endpName, err = we.generateEndpointName(args.Hostname, args.EndpointID)
	if err != nil {
		return errors.Trace(err)
	}

	we.Lock()
	defer we.Unlock()

	return etcd.RetryTimedOut(func() error {
		if err := we.delete(endpName, args.Hostname); err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); !ok {
				return err
			}
		}
		return nil
	}, 3) //nolint
}

func (we *WorkloadEndpoint) delete(endpName, namespace string) (err error) {
	_, err = we.WorkloadEndpoints().Delete(
		context.Background(),
		namespace,
		endpName,
		libcaliopt.DeleteOptions{},
	)
	return
}

func (we *WorkloadEndpoint) getCalicoWorkloadEndpoint(args types.EndpointArgs) (*libcaliapi.WorkloadEndpoint, error) {
	endpName, err := we.generateEndpointName(args.Hostname, args.EndpointID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ipNets, err := we.convCalicoIPNetworks(args.IPs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	profile, err := we.getProfile()
	if err != nil {
		return nil, errors.Trace(err)
	}

	wep := libcaliapi.NewWorkloadEndpoint()
	wep.Name = endpName
	wep.ObjectMeta.Namespace = args.Hostname
	wep.ObjectMeta.ResourceVersion = args.ResourceVersion
	wep.Spec.Endpoint = args.EndpointID
	wep.Spec.Node = args.Hostname
	wep.Spec.Orchestrator = OrchestratorID
	wep.Spec.Workload = OrchestratorID
	wep.Spec.InterfaceName = args.Device.Name()
	wep.Spec.MAC = args.MAC
	wep.Spec.IPNetworks = append(wep.Spec.IPNetworks, ipNets...)
	wep.Spec.Profiles = we.mergeProfile(args.Profiles, profile)

	return wep, nil
}

func (we *WorkloadEndpoint) mergeProfile(profiles []string, other string) []string {
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

func (we *WorkloadEndpoint) generateEndpointName(hostname, endpointID string) (string, error) {
	var ident = libcalinames.WorkloadEndpointIdentifiers{
		Node:         hostname,
		Orchestrator: OrchestratorID,
		Workload:     OrchestratorID,
		Endpoint:     endpointID,
	}
	return ident.CalculateWorkloadEndpointName(false)
}

func (we *WorkloadEndpoint) convCalicoIPNetworks(ips []meta.IP) ([]string, error) {
	var ipNets = make([]string, len(ips))

	for i, ip := range ips {
		ipv4, _, err := netx.ParseCIDR(ip.CIDR())
		if err != nil {
			return nil, errors.Trace(err)
		}

		ipNets[i] = libcalinet.IPNet{IPNet: net.IPNet{
			IP:   ipv4,
			Mask: net.CIDRMask(net.IPv4len*8, net.IPv4len*8), //nolint
		}}.String()
	}

	return ipNets, nil
}

func (we *WorkloadEndpoint) getProfile() (string, error) {
	var pool, err = we.getIPPool()
	if err != nil {
		return "", errors.Trace(err)
	}
	return pool.ObjectMeta.Name, nil
}

// ConvIPs .
func ConvIPs(cwe *libcaliapi.WorkloadEndpoint) (ips []meta.IP, err error) {
	ips = make([]meta.IP, len(cwe.Spec.IPNetworks))

	for i, cidr := range cwe.Spec.IPNetworks {
		if ips[i], err = ParseCIDR(cidr); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return ips, nil
}
