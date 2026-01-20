package calico

import (
	"context"
	"fmt"

	apiv3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	libcalierr "github.com/projectcalico/calico/libcalico-go/lib/errors"
	libcaliopt "github.com/projectcalico/calico/libcalico-go/lib/options"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
	"github.com/projecteru2/yavirt/pkg/terrors"
)

const (
	policyName = "deny-namespaces"
)

func (d *Driver) CreateNetworkPolicy(ns string) (err error) {
	d.Lock()
	defer d.Unlock()

	cwe := genCalicoNetworkPolicy(ns)
	err = etcd.RetryTimedOut(func() error {
		var created, ce = d.NetworkPolicies().Create(context.Background(), cwe, libcaliopt.SetOptions{})
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

func (d *Driver) DeleteNetworkPolicy(ns string) (err error) {
	d.Lock()
	defer d.Unlock()

	return etcd.RetryTimedOut(func() error {
		_, err := d.NetworkPolicies().Delete(
			context.Background(),
			ns,
			policyName,
			libcaliopt.DeleteOptions{},
		)
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); !ok {
				return err
			}
		}
		return nil
	}, 3)
}

func (d *Driver) GetNetworkPolicy(ns string) (cwe *apiv3.NetworkPolicy, err error) {
	d.Lock()
	defer d.Unlock()

	etcd.RetryTimedOut(func() error { //nolint
		cwe, err = d.NetworkPolicies().Get(context.Background(), ns, policyName, libcaliopt.GetOptions{})
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); ok { //nolint
				err = errors.Wrapf(terrors.ErrCalicoEndpointNotExists, "%s on %s", ns, policyName)
			}
		}

		return err
	}, 3)
	return
}

// apiVersion: projectcalico.org/v3
// kind: NetworkPolicy
// metadata:
//
//	name: allow-pool2
//	namespace: testpool2
//
// spec:
//
//	types:
//	  - Ingress
//	  - Egress
//	ingress:
//	  - action: Allow
//	    source:
//	      selector: projectcalico.org/namespace == 'testpool2'
//	  - action: Allow
//	    source:
//	      notNets:
//	        - '10.0.0.0/8'
//	egress:
//	  - action: Allow
//	    destination:
//	      selector: projectcalico.org/namespace == 'testpool2'
//	  - action: Allow
//	    destination:
//	      notNets:
//	        - '10.0.0.0/8'
func genCalicoNetworkPolicy(ns string) *apiv3.NetworkPolicy {
	p := apiv3.NewNetworkPolicy()
	p.Name = policyName

	p.Namespace = ns
	p.Spec.Types = []apiv3.PolicyType{apiv3.PolicyTypeIngress, apiv3.PolicyTypeEgress}
	p.Spec.Ingress = []apiv3.Rule{
		{
			Action: apiv3.Allow,
			Source: apiv3.EntityRule{
				Selector: fmt.Sprintf("projectcalico.org/namespace == '%s'", ns),
			},
		},
		{
			Action: apiv3.Allow,
			Source: apiv3.EntityRule{
				NotNets: []string{
					"10.0.0.0/8",
					"192.168.0.0/16",
				},
			},
		},
	}
	p.Spec.Egress = []apiv3.Rule{
		{
			Action: apiv3.Allow,
			Destination: apiv3.EntityRule{
				Selector: fmt.Sprintf("projectcalico.org/namespace == '%s'", ns),
			},
		},
		{
			Action: apiv3.Allow,
			Destination: apiv3.EntityRule{
				NotNets: []string{
					"10.0.0.0/8",
					"192.168.0.0/16",
				},
			},
		},
	}
	return p
}
