package server

import (
	"os"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/netx"
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

	if err := svc.caliHandler.InitGateway(configs.Conf.CalicoGatewayName); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (svc *Service) setupCalicoHandler() error {
	cali, err := calico.NewDriver(configs.Conf.CalicoConfigFile, configs.Conf.CalicoPoolNames)
	if err != nil {
		return errors.Trace(err)
	}

	dev, err := device.New()
	if err != nil {
		return errors.Trace(err)
	}

	outboundIP, err := netx.GetOutboundIP(configs.Conf.CoreAddr)
	if err != nil {
		return errors.Trace(err)
	}

	svc.caliHandler = calihandler.New(dev, cali, configs.Conf.CalicoPoolNames, outboundIP)

	return nil
}

func (svc *Service) couldSetupCalico() bool {
	var env = configs.Conf.CalicoETCDEnv
	return len(configs.Conf.CalicoConfigFile) > 0 || len(os.Getenv(env)) > 0
}
