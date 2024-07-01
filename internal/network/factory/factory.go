package factory

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/network"
	"github.com/projecteru2/yavirt/internal/network/drivers/calico"
	cniCalico "github.com/projecteru2/yavirt/internal/network/drivers/cni/calico"
	"github.com/projecteru2/yavirt/internal/network/drivers/fake"
	"github.com/projecteru2/yavirt/internal/network/drivers/ovn"
	"github.com/projecteru2/yavirt/internal/network/drivers/vlan"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gF *Factory
)

type Factory struct {
	Config          *configs.NetworkConfig
	networkHandlers map[string]network.Driver
}

func Setup(cfg *configs.NetworkConfig) (err error) {
	gF, err = New(cfg)
	return err
}

func CheckHealth(ctx context.Context) error {
	for _, d := range gF.networkHandlers {
		if err := d.CheckHealth(ctx); err != nil {
			return err
		}
	}
	return nil
}

func GetMetricsCollectors() (ans []prometheus.Collector) {
	for _, d := range gF.networkHandlers {
		col := d.GetMetricsCollector()
		if col == nil {
			continue
		}
		ans = append(ans, col)
	}
	return ans
}

func GetDriver(mode string) network.Driver {
	return gF.networkHandlers[mode]
}

func ListDrivers() map[string]network.Driver {
	return gF.networkHandlers
}

func New(cfg *configs.NetworkConfig) (*Factory, error) {
	f := &Factory{
		Config: cfg,
		networkHandlers: map[string]network.Driver{
			network.FakeMode: &fake.Driver{},
		},
	}
	err := f.setupDrivers()
	return f, err
}

func (f *Factory) setupDrivers() error {
	cfg := f.Config
	for _, mode := range cfg.Modes {
		switch mode {
		case network.CalicoMode:
			if err := f.setupCalico(); err != nil {
				return errors.Wrap(err, "")
			}
		case network.CalicoCNIMode:
			if err := f.setupCalicoCNI(); err != nil {
				return errors.Wrap(err, "")
			}
		case network.VlanMode:
			if err := f.setupVlan(); err != nil {
				return errors.Wrap(err, "")
			}
		case network.OVNMode:
			if err := f.setupOVN(); err != nil {
				return errors.Wrap(err, "")
			}
		default:
			return errors.Newf("invalid network mode: %s", mode)
		}
	}
	return nil
}

func (f *Factory) setupCalicoCNI() error {
	cali, err := calico.NewDriver(&f.Config.Calico)
	if err != nil {
		return errors.Wrap(err, "")
	}

	driver, err := cniCalico.NewDriver(&f.Config.CNI, cali)
	if err != nil {
		return errors.Wrap(err, "")
	}
	f.networkHandlers[network.CalicoCNIMode] = driver
	return nil
}

func (f *Factory) setupVlan() error { //nolint
	cfg := f.Config.Vlan
	f.networkHandlers[network.VlanMode] = vlan.New(int64(cfg.Subnet))
	return nil
}

func (f *Factory) setupOVN() (err error) {
	cfg := &f.Config.OVN
	f.networkHandlers[network.OVNMode], err = ovn.NewDriver(cfg)
	return
}

func (f *Factory) setupCalico() error {
	cali, err := calico.NewDriver(&f.Config.Calico)
	if err != nil {
		return errors.Wrap(err, "")
	}

	// if err := svc.caliHandler.InitGateway(f.Config.Calico.GatewayName); err != nil {
	// 	return errors.Wrap(err, "")
	// }

	f.networkHandlers[network.CalicoMode] = cali
	return nil
}
