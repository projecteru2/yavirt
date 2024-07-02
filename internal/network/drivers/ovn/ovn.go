package ovn

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/model"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/network/types"
	netutils "github.com/projecteru2/yavirt/internal/network/utils"
	"github.com/projecteru2/yavirt/internal/utils"
	"github.com/shirou/gopsutil/process"
)

const (
	ovnMTU = 1442
)

type Driver struct {
	sync.Mutex
	mCol             *MetricsCollector
	cfg              *configs.OVNConfig
	nbClientDBModel  model.ClientDBModel
	nbCli            client.Client
	ovsClientDBModel model.ClientDBModel
	ovsCli           client.Client
}

func NewDriver(cfg *configs.OVNConfig) (*Driver, error) {
	return &Driver{cfg: cfg, mCol: &MetricsCollector{}}, nil
}

func (d *Driver) CheckHealth(_ context.Context) (err error) {
	defer func() {
		if err != nil {
			d.mCol.healthy.Store(false)
		} else {
			d.mCol.healthy.Store(true)
		}
	}()
	processes, err := process.Processes()
	if err != nil {
		return err
	}
	binaries := []string{
		"ovn-controller",
		"ovsdb-server",
		"ovs-vswitchd",
	}
	for _, name := range binaries {
		if !utils.PSContains([]string{name}, processes) {
			return errors.Newf("%s is not running", name)
		}
	}

	return nil
}

func (d *Driver) QueryIPs(args meta.IPNets) ([]meta.IP, error) {
	if len(args) == 0 {
		return nil, nil
	}
	ans := make([]meta.IP, 0, len(args))
	for _, ipn := range args {
		ip, err := NewIP("", ipn.CIDR())
		if err != nil {
			return nil, err
		}
		ans = append(ans, ip)
	}
	return ans, nil
}

func (d *Driver) CreateEndpointNetwork(args types.EndpointArgs) (types.EndpointArgs, func() error, error) {
	var (
		err error
		ls  *LogicalSwitch
		lsp *LogicalSwitchPort
	)

	if args.DevName, err = netutils.GenDevName(configs.Conf.Network.OVN.IFNamePrefix); err != nil {
		return types.EndpointArgs{}, nil, err
	}
	// get LogicalSwitch
	if args.OVN.LogicalSwitchUUID != "" {
		if ls, err = d.getLogicalSwitch(args.OVN.LogicalSwitchUUID); err != nil {
			return types.EndpointArgs{}, nil, err
		}
	} else {
		lsList, err := d.getLogicalSwitchByName(args.OVN.LogicalSwitchName)
		if err != nil {
			return types.EndpointArgs{}, nil, err
		}
		switch len(lsList) {
		case 1:
			ls = lsList[0]
		case 0:
			return types.EndpointArgs{}, nil, fmt.Errorf("logical switch %s not found", args.OVN.LogicalSwitchName)
		default:
			return types.EndpointArgs{}, nil, fmt.Errorf("multiple logical switch %s found", args.OVN.LogicalSwitchName)
		}
	}
	// create LogicalSwitchPort if necessary and then get it
	if args.OVN.LogicalSwitchPortName != "" {
		if lsp, err = d.getLogicalSwitchPortByName(args.OVN.LogicalSwitchPortName); err != nil {
			return types.EndpointArgs{}, nil, err
		}
	} else {
		lspUUID, err := d.createLogicalSwitchPort(&args)
		if err != nil {
			return types.EndpointArgs{}, nil, err
		}
		if lsp, err = d.getLogicalSwitchPort(lspUUID); err != nil {
			return types.EndpointArgs{}, nil, err
		}
	}
	subnet := ls.Config["subnet"]
	addrs := strings.Split(*lsp.DynamicAddresses, " ")
	mac := addrs[0]
	ipv4 := addrs[1]
	args.MAC = mac
	args.MTU = ovnMTU
	ip, err := NewIP(ipv4, subnet)
	if err != nil {
		return types.EndpointArgs{}, nil, err
	}
	args.IPs = append(args.IPs, ip)
	return args, nil, nil
}

func (d *Driver) JoinEndpointNetwork(args types.EndpointArgs) (func() error, error) {
	lspName := args.OVN.LogicalSwitchPortName
	if lspName == "" {
		lspName = LSPName(args.GuestID)
	}
	err := d.setExternalID(args.DevName, "iface-id", lspName)
	return nil, err
}

func (d *Driver) DeleteEndpointNetwork(args types.EndpointArgs) error {
	// lsp is provided by other component, so we ignore deleting lsp here
	if args.OVN.LogicalSwitchPortName != "" {
		return nil
	}
	return d.deleteLogicalSwitchPort(&args)
}

func (d *Driver) CreateNetworkPolicy(string) error {
	return nil
}

func (d *Driver) DeleteNetworkPolicy(string) error {
	return nil
}

func LSPName(guestID string) string {
	return fmt.Sprintf("lsp-%s", guestID)
}
