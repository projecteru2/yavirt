package yavirtd

import (
	"os"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/netx"
	"github.com/projecteru2/yavirt/vnet"
	"github.com/projecteru2/yavirt/vnet/calico"
	"github.com/projecteru2/yavirt/vnet/device"
	calihandler "github.com/projecteru2/yavirt/vnet/handler/calico"
)

func (svc *Service) setupCalico() error {
	if !svc.couldSetupCalico() {
		if svc.Host.NetworkMode == vnet.NetworkCalico {
			return errors.Annotatef(errors.ErrInvalidValue, "invalid Calico config")
		}
		return nil
	}

	if err := svc.setupCalicoHandler(); err != nil {
		return errors.Trace(err)
	}

	if err := svc.caliHandler.InitGateway(config.Conf.CalicoGatewayName); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (svc *Service) setupCalicoHandler() error {
	cali, err := calico.NewDriver(config.Conf.CalicoConfigFile, config.Conf.CalicoPoolNames)
	if err != nil {
		return errors.Trace(err)
	}

	dev, err := device.New()
	if err != nil {
		return errors.Trace(err)
	}

	outboundIP, err := netx.GetOutboundIP(config.Conf.CoreAddr)
	if err != nil {
		return errors.Trace(err)
	}

	svc.caliHandler = calihandler.New(dev, cali, config.Conf.CalicoPoolNames, outboundIP)

	return nil
}

func (svc *Service) couldSetupCalico() bool {
	var env = config.Conf.CalicoETCDEnv
	return len(config.Conf.CalicoConfigFile) > 0 || len(os.Getenv(env)) > 0
}
