package calico

import (
	"context"
	"net"
	"strings"
	"sync"

	calitype "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	libcaliapi "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/network/utils/device"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/terrors"

	"github.com/cockroachdb/errors"
)

// Driver .
type Driver struct {
	sync.Mutex

	clientv3.Interface

	mCol *MetricsCollector
	dev  *device.Driver

	gateway                 *device.Dummy
	gatewayWorkloadEndpoint *libcaliapi.WorkloadEndpoint

	nodename  string
	hostIP    string
	poolNames map[string]struct{}
	dhcp      *DHCPServer
}

// NewDriver .
func NewDriver(cfg *configs.CalicoConfig) (*Driver, error) {
	configFile, poolNames := cfg.ConfigFile, cfg.PoolNames
	dev, err := device.New()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	hostIP, err := netx.GetOutboundIP("8.8.8.8:53")
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	caliConf, err := apiconfig.LoadClientConfig(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	cali, err := clientv3.New(*caliConf)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var driver = &Driver{
		Interface: cali,
		mCol:      &MetricsCollector{},
		nodename:  cfg.Nodename,
		dev:       dev,
		hostIP:    hostIP,
		poolNames: map[string]struct{}{},
	}

	for _, pn := range poolNames {
		driver.poolNames[pn] = struct{}{}
	}

	return driver, nil
}

func (d *Driver) CheckHealth(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			d.mCol.healthy.Store(false)
		} else {
			d.mCol.healthy.Store(true)
		}
	}()
	n, err := d.Nodes().Get(ctx, d.nodename, options.GetOptions{})
	if err != nil {
		return err
	}
	if n == nil {
		return errors.Newf("calico node %s not found", d.nodename)
	}
	if err := CheckNodeStatus(); err != nil {
		return err
	}
	return nil
}

func (d *Driver) InitDHCP() error {
	logger := log.WithFunc("calico.initDHCP")
	gwIP := net.ParseIP("169.254.1.1")
	dhcpSrv := NewDHCPServer(gwIP)
	weps, err := d.ListWEP()
	if err != nil {
		return err
	}
	for _, wep := range weps {
		iface := wep.Spec.InterfaceName
		if len(wep.Spec.IPNetworks) == 0 {
			continue
		}
		ip, ipNet, err := net.ParseCIDR(wep.Spec.IPNetworks[0])
		if err != nil {
			logger.Errorf(context.TODO(), err, "failed to parse cidr: %s", wep.Spec.IPNetworks[0])
			continue
		}
		if err := dhcpSrv.AddInterface(iface, ip, ipNet); err != nil {
			logger.Errorf(context.TODO(), err, "failed to add interface to dhcp.")
		}
	}
	d.dhcp = dhcpSrv
	return nil
}

func (d *Driver) getIPPool(poolName string) (pool *calitype.IPPool, err error) {
	if poolName != "" {
		return d.IPPools().Get(context.Background(), poolName, options.GetOptions{})
	}
	pools, err := d.IPPools().List(context.Background(), options.ListOptions{})
	switch {
	case err != nil:
		return pool, errors.Wrap(err, "")
	case len(pools.Items) < 1:
		return pool, errors.Wrap(terrors.ErrCalicoPoolNotExists, "")
	}

	if len(d.poolNames) < 1 {
		return &pools.Items[0], nil
	}

	for _, p := range pools.Items {
		if _, exists := d.poolNames[p.Name]; exists {
			return &p, nil
		}
	}

	return pool, errors.Wrapf(terrors.ErrCalicoPoolNotExists, "no such pool names: %s", d.poolNamesStr())
}

func (d *Driver) poolNamesStr() string {
	names := d.PoolNames()
	return strings.Join(names, ", ")
}

// Ipam .
func (d *Driver) Ipam() *Ipam {
	return newIpam(d)
}
