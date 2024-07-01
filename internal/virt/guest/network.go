package guest

import (
	"context"
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/florianl/go-tc"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/network"
	networkFactory "github.com/projecteru2/yavirt/internal/network/factory"
	"github.com/projecteru2/yavirt/internal/network/types"
	interutils "github.com/projecteru2/yavirt/internal/utils"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	// for compatibility
	calicoIPPoolLabelKey = "calico/ippool"
	calicoNSLabelKey     = "calico/namespace"
)

// DisconnectExtraNetwork .
func (g *Guest) DisconnectExtraNetwork(_ string) error {
	// todo
	return nil
}

// ConnectExtraNetwork .
func (g *Guest) ConnectExtraNetwork(_, _ string) (ip meta.IP, err error) {
	// todo
	return
}

func (g *Guest) CreateNetwork(ctx context.Context) (err error) {
	if g.MAC, err = utils.QemuMAC(); err != nil {
		return errors.Wrap(err, "")
	}

	if _, err = g.createEthernet(); err != nil {
		return errors.Wrap(err, "")
	}
	rl := interutils.GetRollbackListFromContext(ctx)
	if rl != nil {
		rl.Append(func() error {
			return g.botOperate(func(bot Bot) error { //nolint:revive
				return g.DeleteNetwork()
			}, true)
		}, "delete network")
	}
	return nil
}

func (g *Guest) getEndpointArgs() (types.EndpointArgs, error) {
	hn := configs.Hostname()
	args := types.EndpointArgs{
		GuestID:    g.ID,
		MAC:        g.MAC,
		MTU:        g.MTU,
		Hostname:   hn,
		EndpointID: g.EndpointID,
		IPs:        g.IPs,
		DevName:    g.NetworkPair,
	}
	// just for compatibility
	if args.MTU == 0 {
		args.MTU = 1500
	}
	switch g.NetworkMode {
	case network.OVNMode:
		var ovnArgs types.OVNArgs
		rawJSON := g.JSONLabels[network.OVNLabelKey]
		if rawJSON == "" {
			return args, errors.Errorf("ovn args not found")
		}
		if err := json.Unmarshal([]byte(rawJSON), &ovnArgs); err != nil {
			return args, errors.Wrap(err, "")
		}
		args.OVN = ovnArgs
	case network.CalicoMode:
		var calicoArgs types.CalicoArgs
		rawJSON := g.JSONLabels[network.CalicoLabelKey]
		if rawJSON != "" {
			if err := json.Unmarshal([]byte(rawJSON), &calicoArgs); err != nil {
				return args, errors.Wrap(err, "")
			}
		} else {
			calicoArgs.IPPool = g.JSONLabels[calicoIPPoolLabelKey]
			calicoArgs.Namespace = g.JSONLabels[calicoNSLabelKey]
		}
		if calicoArgs.Namespace == "" {
			calicoArgs.Namespace = hn
		}
		args.Calico = calicoArgs
	case network.CalicoCNIMode:
		// TODO implement CNI
		var cniArgs types.CNIArgs
		args.CNI = cniArgs
	case network.VlanMode:
		// TODO
		var vlanArgs types.VlanArgs
		args.Vlan = vlanArgs
	case network.FakeMode:
		// do nothing
	default:
		return args, errors.Errorf("unsupported network mode %s", g.NetworkMode)
	}
	return args, nil
}

// createEthernet .
func (g *Guest) createEthernet() (rollback func() error, err error) {
	var hand network.Driver
	hand, err = g.NetworkHandler()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	args, err := g.getEndpointArgs()
	if err != nil {
		return nil, err
	}
	var rollCreate func() error
	args, rollCreate, err = hand.CreateEndpointNetwork(args)
	switch {
	case err != nil:
		return nil, errors.Wrap(err, "")
	case args.DevName != "":
		g.NetworkPair = args.DevName
	}

	g.EndpointID = args.EndpointID
	g.MAC = args.MAC
	g.MTU = args.MTU
	g.AppendIPs(args.IPs...)
	return rollCreate, nil
}

func (g *Guest) joinEthernet() (err error) {
	var hand network.Driver
	if hand, err = g.NetworkHandler(); err != nil {
		return errors.Wrap(err, "")
	}

	args, err := g.getEndpointArgs()
	if err != nil {
		return errors.Wrapf(err, "failed to join ethernet")
	}
	_, err = hand.JoinEndpointNetwork(args)

	return
}

// DeleteNetwork .
func (g *Guest) DeleteNetwork() error {
	return g.deleteEthernet()
}

func (g *Guest) deleteEthernet() error {
	hand, err := g.NetworkHandler()
	if err != nil {
		return errors.Wrap(err, "")
	}

	args, err := g.getEndpointArgs()
	if err != nil {
		return errors.Wrapf(err, "failed to delete ethernet")
	}
	if err := hand.DeleteEndpointNetwork(args); err != nil {
		return errors.Wrap(err, "")
	}

	g.IPs = models.IPs{}
	g.IPNets = meta.IPNets{}

	return nil
}

func (g *Guest) loadExtraNetworks() error {
	// todo
	return nil
}

// NetworkHandler .
func (g *Guest) NetworkHandler() (network.Driver, error) {
	d := networkFactory.GetDriver(g.NetworkMode)
	if d == nil {
		return nil, errors.Wrapf(terrors.ErrUnknownNetworkDriver, "guest: %s, networkMode: %s", g.ID, g.NetworkMode)
	}
	return d, nil
}

func (g *Guest) limitBandwidth() error {
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		log.Errorf(context.TODO(), err, "[limitBandwidth] could not open rtnetlink socket")
		return err
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			log.Errorf(context.TODO(), err, "[limitBandwidth] could not close rtnetlink socket")
		}
	}()

	return nil
}
