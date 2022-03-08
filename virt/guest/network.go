package guest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/types/current"

	"github.com/projecteru2/yavirt/config"
	"github.com/projecteru2/yavirt/errors"
	"github.com/projecteru2/yavirt/log"
	"github.com/projecteru2/yavirt/meta"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/sh"
	"github.com/projecteru2/yavirt/util"
	"github.com/projecteru2/yavirt/vnet"
	calinet "github.com/projecteru2/yavirt/vnet/calico"
	"github.com/projecteru2/yavirt/vnet/handler"
	calihandler "github.com/projecteru2/yavirt/vnet/handler/calico"
	vlanhandler "github.com/projecteru2/yavirt/vnet/handler/vlan"
	"github.com/projecteru2/yavirt/vnet/types"
)

const (
	cniCmdAdd = "ADD"
	cniCmdDel = "DEL"
)

// DisconnectExtraNetwork .
func (g *Guest) DisconnectExtraNetwork(network string) error {
	// todo
	return nil
}

// ConnectExtraNetwork .
func (g *Guest) ConnectExtraNetwork(network, ipv4 string) (ip meta.IP, err error) {
	// todo
	return
}

// CreateEthernet .
func (g *Guest) CreateEthernet() (rollback func() error, err error) {
	if g.EnabledCalicoCNI {
		return g.calicoCNICreate()
	}

	var ip meta.IP
	if ip, err = g.assignIP(); err != nil {
		return nil, errors.Trace(err)
	}

	var rollbackIP = func() error {
		return g.releaseIPs(ip)
	}

	defer func() {
		if err != nil {
			if re := rollbackIP(); re != nil {
				err = errors.Wrap(err, re)
			}
		}
	}()

	var rollbackEndpoint func() error
	if rollbackEndpoint, err = g.createEndpoint(); err != nil {
		return nil, errors.Trace(err)
	}

	return func() error {
		var err = errors.Errorf("rollback network for %s", ip)

		if re := rollbackEndpoint(); re != nil {
			return errors.Wrap(err, re)
		}

		if re := rollbackIP(); re != nil {
			return errors.Wrap(err, re)
		}

		return err
	}, nil
}

func (g *Guest) createEndpoint() (rollback func() error, err error) {
	var hn string
	hn, err = util.Hostname()
	if err != nil {
		return nil, errors.Trace(err)
	}

	var hand handler.Handler
	hand, err = g.NetworkHandler(g.Host)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var args types.EndpointArgs
	args.IPs = g.IPs
	args.MAC = g.MAC
	args.Hostname = hn

	var rollCreate func()
	args, rollCreate, err = hand.CreateEndpointNetwork(args)
	switch {
	case err != nil:
		return nil, errors.Trace(err)
	case args.Device != nil:
		g.NetworkPair = args.Device.Name()
	}

	g.EndpointID = args.EndpointID

	var unjoin func()
	if unjoin, err = hand.JoinEndpointNetwork(args); err != nil {
		rollCreate()
		return nil, errors.Trace(err)
	}

	rollback = func() error {
		unjoin()
		rollCreate()
		return nil
	}

	return rollback, nil
}

func (g *Guest) joinEthernet() (err error) {
	if g.EnabledCalicoCNI {
		_, _, err = g.calicoCNIAdd(false)
		return errors.Trace(err)
	}

	var hand handler.Handler
	if hand, err = g.NetworkHandler(g.Host); err != nil {
		return errors.Trace(err)
	}

	var args types.EndpointArgs
	args.IPs = g.IPs
	args.MAC = g.MAC
	args.EndpointID = g.EndpointID

	if args.Hostname, err = util.Hostname(); err != nil {
		return errors.Trace(err)
	}

	if args.Device, err = hand.GetEndpointDevice(g.NetworkPair); err != nil {
		return errors.Trace(err)
	}

	_, err = hand.JoinEndpointNetwork(args)

	return
}

func (g *Guest) assignIP() (meta.IP, error) {
	hand, err := g.NetworkHandler(g.Host)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ip, err := hand.AssignIP()
	if err != nil {
		return nil, errors.Trace(err)
	}

	g.AppendIPs(ip)

	return ip, nil
}

// DeleteNetwork .
func (g *Guest) DeleteNetwork() error {
	return g.deleteEthernet()
}

func (g *Guest) deleteEthernet() error {
	if g.EnabledCalicoCNI {
		return g.calicoCNIDel()
	}

	hn, err := util.Hostname()
	if err != nil {
		return errors.Trace(err)
	}

	hand, err := g.NetworkHandler(g.Host)
	if err != nil {
		return errors.Trace(err)
	}

	var args = types.EndpointArgs{}
	args.EndpointID = g.EndpointID
	args.Hostname = hn

	if err := hand.DeleteEndpointNetwork(args); err != nil {
		return errors.Trace(err)
	}

	if err := g.releaseIPs(g.IPs...); err != nil {
		return errors.Trace(err)
	}

	g.IPs = model.IPs{}
	g.IPNets = meta.IPNets{}

	return nil
}

func (g *Guest) loadExtraNetworks() error {
	// todo
	return nil
}

func (g *Guest) releaseIPs(ips ...meta.IP) error {
	var hand, err = g.NetworkHandler(g.Host)
	if err != nil {
		return errors.Trace(err)
	}
	return hand.ReleaseIPs(ips...)
}

// NetworkHandler .
func (g *Guest) NetworkHandler(host *model.Host) (handler.Handler, error) {
	switch g.NetworkMode {
	case vnet.NetworkCalico:
		return g.ctx.CalicoHandler()

	case vnet.NetworkVlan:
		fallthrough
	case "":
		return vlanhandler.New(g.ID, host.Subnet), nil

	default:
		return nil, errors.Annotatef(errors.ErrInvalidValue, "invalid network: %s", g.NetworkMode)
	}
}

func (g *Guest) calicoCNIDel() error {
	env := g.makeCNIEnv()
	env["CNI_COMMAND"] = cniCmdDel

	dat, err := g.readCNIConfig()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = execCNIPlugin(env, bytes.NewBuffer(dat), config.Conf.CNIPluginPath)
	return err
}

func (g *Guest) calicoCNICreate() (func() error, error) {
	endpointID, err := util.UUIDStr()
	if err != nil {
		return nil, errors.Trace(err)
	}
	endpointID = strings.ReplaceAll(endpointID, "-", "")

	g.EndpointID = endpointID
	g.NetworkPair = "yap" + g.EndpointID[:util.Min(12, len(g.EndpointID))] //nolint

	stdout, execDel, err := g.calicoCNIAdd(true)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err := g.populateIPFromAddResult(stdout); err != nil {
		if de := execDel(); de != nil {
			return nil, errors.Wrap(err, de)
		}
	}

	return execDel, nil
}

func (g *Guest) calicoCNIAdd(needRollback bool) (stdout []byte, rollback func() error, err error) {
	env := g.makeCNIEnv()
	env["CNI_COMMAND"] = cniCmdAdd

	var dat []byte
	if dat, err = g.readCNIConfig(); err != nil {
		return nil, nil, errors.Trace(err)
	}

	if stdout, err = execCNIPlugin(env, bytes.NewBuffer(dat), config.Conf.CNIPluginPath); err != nil {
		return nil, nil, errors.Trace(err)
	}

	execDel := func() error {
		env["CNI_COMMAND"] = cniCmdDel
		_, err := execCNIPlugin(env, bytes.NewBuffer(dat), config.Conf.CNIPluginPath)
		return err
	}

	defer func() {
		if err != nil && needRollback {
			if de := execDel(); de != nil {
				err = errors.Wrap(err, de)
			}
			execDel = nil
		}
	}()

	hand, err := g.calicoHandler()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	// Refreshes gateway for non-Calico-CNI operations.
	if err = hand.RefreshGateway(); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return stdout, execDel, nil
}

func (g *Guest) populateIPFromAddResult(dat []byte) error {
	var result current.Result
	if err := json.Unmarshal(dat, &result); err != nil {
		return errors.Trace(err)
	}
	if len(result.IPs) < 1 {
		return errors.Trace(errors.ErrIPIsnotAssigned)
	}

	hand, err := g.calicoHandler()
	if err != nil {
		return errors.Trace(err)
	}

	for _, ipConf := range result.IPs {
		ip, err := calinet.ParseCIDR(ipConf.Address.String())
		if err != nil {
			return errors.Trace(err)
		}

		gwip, err := hand.GetGatewayIP(ip)
		if err != nil {
			return errors.Trace(err)
		}

		ip.BindGatewayIPNet(gwip.IPNetwork())

		g.AppendIPs(ip)
	}

	return nil
}

func (g *Guest) readCNIConfig() ([]byte, error) {
	// TODO: follows the CNI policy, rather than hard code absolute path here.
	return ioutil.ReadFile(config.Conf.CNIConfigPath)
}

func (g *Guest) makeCNIEnv() map[string]string {
	return map[string]string{
		"CNI_CONTAINERID": g.ID,
		"CNI_ARGS":        "IgnoreUnknown=1;MAC=" + g.MAC,
		"CNI_IFNAME":      g.NetworkPair,
		"CNI_PATH":        filepath.Dir(config.Conf.CNIPluginPath),
		"CNI_NETNS":       "yap",
	}
}

func (g *Guest) calicoHandler() (*calihandler.Handler, error) {
	raw, err := g.NetworkHandler(g.Host)
	if err != nil {
		return nil, errors.Trace(err)
	}

	hand, ok := raw.(*calihandler.Handler)
	if !ok {
		return nil, errors.Annotatef(errors.ErrInvalidValue, "invalid *calihandler.Handler: %v", raw)
	}

	return hand, nil
}

func execCNIPlugin(env map[string]string, stdin io.Reader, plugin string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*8) //nolint
	defer cancel()

	log.Debugf("CNI Plugin env: %v", env)
	so, se, err := sh.ExecInOut(ctx, env, stdin, plugin)

	if err != nil {
		err = errors.Annotatef(err, "Failed to exec %s with %v: %s: %s", plugin, string(so), string(se))
	}

	return so, err
}
