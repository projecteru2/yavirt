package bison

import (
	"errors"
	"testing"

	"github.com/alphadose/haxmap"
	"github.com/projecteru2/yavirt/internal/utils/notify"
	"github.com/projecteru2/yavirt/internal/utils/notify/services/all"
	"github.com/projecteru2/yavirt/internal/utils/notify/services/dingding"
	"github.com/projecteru2/yavirt/internal/utils/notify/services/mail"
)

var bs *Manager

type Config struct {
	Type     string          `toml:"type"`
	DingDing dingding.Config `toml:"dingding"`
	Mail     mail.Config     `toml:"mail"`
	All      all.Config      `toml:"all"`
}

type Manager struct {
	config   Config
	services *haxmap.Map[string, notify.Service]
}

func loadService(cfg *Config, ty string) (svc notify.Service, err error) {
	switch ty {
	case "dingding":
		svc, err = dingding.New(cfg.DingDing)
	case "mail":
		svc = mail.New(cfg.Mail)
	default:
		return nil, errors.New("type not supported")
	}
	return
}

func New(cfg *Config) (mgr *Manager, err error) {
	mgr = &Manager{
		config:   *cfg,
		services: haxmap.New[string, notify.Service](),
	}
	var svc notify.Service
	if cfg.Type == "all" {
		var lst []notify.Service
		for _, ty := range cfg.All.Types {
			svc, err := loadService(cfg, ty)
			if err != nil {
				return nil, err
			}
			lst = append(lst, svc)
		}
		svc = all.New(lst)
	} else {
		svc, err = loadService(cfg, cfg.Type)
		if err != nil {
			return nil, err
		}
	}
	mgr.services.Set(cfg.Type, svc)
	return mgr, nil
}

func (mgr *Manager) GetService() notify.Service {
	svc, ok := mgr.services.Get(mgr.config.Type)
	if !ok {
		return nil
	}
	return svc
}

func Setup(cfg *Config, t *testing.T) error {
	var err error
	if t != nil {
		bs = &Manager{
			config: Config{
				Type: "test",
			},
			services: haxmap.New[string, notify.Service](),
		}
		return nil
	}
	bs, err = New(cfg)
	return err
}

func GetService() notify.Service {
	return bs.GetService()
}
