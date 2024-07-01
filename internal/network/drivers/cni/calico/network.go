package calico

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/errors"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/projecteru2/core/log"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/meta"
	calihandler "github.com/projecteru2/yavirt/internal/network/drivers/calico"
	"github.com/projecteru2/yavirt/internal/network/types"
	netutils "github.com/projecteru2/yavirt/internal/network/utils"
	"github.com/projecteru2/yavirt/pkg/sh"
	"github.com/projecteru2/yavirt/pkg/terrors"
	"github.com/projecteru2/yavirt/pkg/utils"
)

const (
	cniCmdAdd            = "ADD"
	cniCmdDel            = "DEL"
	calicoIPPoolLabelKey = "calico/ippool"
	calicoNSLabelKey     = "calico/namespace"
)

type Driver struct {
	*calihandler.Driver
	cfg *configs.CNIConfig
}

func NewDriver(cfg *configs.CNIConfig, cali *calihandler.Driver) (*Driver, error) {
	return &Driver{
		Driver: cali,
		cfg:    cfg,
	}, nil
}

func (d *Driver) QueryIPs(meta.IPNets) ([]meta.IP, error) {
	return nil, nil
}

func (d *Driver) CreateEndpointNetwork(args types.EndpointArgs) (types.EndpointArgs, func() error, error) {
	var err error
	if args.EndpointID, err = netutils.GenEndpointID(); err != nil {
		return args, nil, errors.Wrap(err, "")
	}

	stdout, execDel, err := d.calicoCNIAdd(&args, true)
	if err != nil {
		return args, nil, errors.Wrap(err, "")
	}

	if err := d.populateIPFromAddResult(stdout, &args); err != nil {
		if de := execDel(); de != nil {
			return args, nil, errors.CombineErrors(err, de)
		}
	}

	return args, execDel, nil
}

func (d *Driver) JoinEndpointNetwork(args types.EndpointArgs) (func() error, error) {
	_, _, err := d.calicoCNIAdd(&args, false)
	return nil, errors.Wrap(err, "")
}

func (d *Driver) DeleteEndpointNetwork(args types.EndpointArgs) error {
	return d.calicoCNIDel(&args)
}

func (d *Driver) calicoCNIDel(args *types.EndpointArgs) error {
	env := d.makeCNIEnv(args)
	env["CNI_COMMAND"] = cniCmdDel

	dat, err := d.readCNIConfig()
	if err != nil {
		return errors.Wrap(err, "")
	}

	_, err = execCNIPlugin(env, bytes.NewBuffer(dat), d.cfg.PluginPath)
	return err
}

func (d *Driver) calicoCNIAdd(args *types.EndpointArgs, needRollback bool) (stdout []byte, rollback func() error, err error) {
	env := d.makeCNIEnv(args)
	env["CNI_COMMAND"] = cniCmdAdd

	var dat []byte
	if dat, err = d.readCNIConfig(); err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	if stdout, err = execCNIPlugin(env, bytes.NewBuffer(dat), d.cfg.PluginPath); err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	execDel := func() error {
		env["CNI_COMMAND"] = cniCmdDel
		_, err := execCNIPlugin(env, bytes.NewBuffer(dat), d.cfg.PluginPath)
		return err
	}

	defer func() {
		if err != nil && needRollback {
			if de := execDel(); de != nil {
				err = errors.CombineErrors(err, de)
			}
			execDel = nil
		}
	}()

	// Refreshes gateway for non-Calico-CNI operations.
	if err = d.RefreshGateway(); err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	return stdout, execDel, nil
}

func (d *Driver) populateIPFromAddResult(dat []byte, args *types.EndpointArgs) error {
	var result current.Result
	if err := json.Unmarshal(dat, &result); err != nil {
		return errors.Wrap(err, "")
	}
	if len(result.IPs) < 1 {
		return errors.Wrap(terrors.ErrIPIsnotAssigned, "")
	}

	for _, ipConf := range result.IPs {
		ip, err := calihandler.ParseCIDR(ipConf.Address.String())
		if err != nil {
			return errors.Wrap(err, "")
		}

		gwip, err := d.GetGatewayIP(ip)
		if err != nil {
			return errors.Wrap(err, "")
		}

		ip.BindGatewayIPNet(gwip.IPNetwork())
		args.IPs = append(args.IPs, ip)
	}

	return nil
}

func (d *Driver) readCNIConfig() ([]byte, error) {
	// TODO: follows the CNI policy, rather than hard code absolute path here.
	return os.ReadFile(d.cfg.ConfigPath)
}

func (d *Driver) makeCNIEnv(args *types.EndpointArgs) map[string]string {
	networkPair := d.cfg.IFNamePrefix + args.EndpointID[:utils.Min(12, len(args.EndpointID))]
	return map[string]string{
		"CNI_CONTAINERID": args.GuestID,
		"CNI_ARGS":        "IgnoreUnknown=1;MAC=" + args.MAC,
		"CNI_IFNAME":      networkPair,
		"CNI_PATH":        filepath.Dir(d.cfg.PluginPath),
		"CNI_NETNS":       "yap",
	}
}

func execCNIPlugin(env map[string]string, stdin io.Reader, plugin string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*8)
	defer cancel()

	log.Debugf(context.TODO(), "CNI Plugin env: %v", env)
	so, se, err := sh.ExecInOut(ctx, env, stdin, plugin)

	if err != nil {
		err = errors.Wrapf(err, "Failed to exec %s with %v: %s: %s", plugin, string(so), string(se))
	}

	return so, err
}
