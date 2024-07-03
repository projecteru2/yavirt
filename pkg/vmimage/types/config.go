package types

import (
	"encoding/base64"
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"
)

type DockerConfig struct {
	Endpoint string `toml:"endpoint" default:"unix:///var/run/docker.sock"`
	Auth     string `toml:"auth"` // in base64
	Prefix   string `toml:"prefix"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type VMIHubConfig struct {
	BaseDir  string `toml:"base_dir"`
	Addr     string `toml:"addr"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type Config struct {
	Type   string       `toml:"type" default:"docker"`
	Docker DockerConfig `toml:"docker"`
	VMIHub VMIHubConfig `toml:"vmihub"`
}

func (cfg *Config) CheckAndRefine() error {
	switch cfg.Type {
	case "docker":
		if cfg.Docker.Username == "" || cfg.Docker.Password == "" {
			return errors.New("docker's username or password should not be empty")
		}
		auth := map[string]string{
			"username": cfg.Docker.Username,
			"password": cfg.Docker.Password,
		}
		authBytes, _ := json.Marshal(auth)
		cfg.Docker.Auth = base64.StdEncoding.EncodeToString(authBytes)
	case "vmihub":
		if cfg.VMIHub.Username == "" || cfg.VMIHub.Password == "" {
			return errors.New("ImageHub's username or password should not be empty")
		}
		if cfg.VMIHub.Addr == "" {
			return errors.New("ImageHub's address shouldn't be empty")
		}
		u, err := url.Parse(cfg.VMIHub.Addr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %s", cfg.VMIHub.Addr)
		}
		if u.Scheme == "" || u.Host == "" {
			return errors.New("invalid image hub addr")
		}
	case "mock":
		return nil
	default:
		return errors.New("unknown image hub type")
	}
	return nil
}
