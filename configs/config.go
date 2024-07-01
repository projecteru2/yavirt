package configs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dustin/go-humanize"
	"github.com/mcuadros/go-defaults"
	"github.com/projecteru2/yavirt/internal/utils/notify/bison"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/utils"
	"github.com/urfave/cli/v2"

	coretypes "github.com/projecteru2/core/types"
	vmitypes "github.com/yuyang0/vmimage/types"
)

var (
	Conf = newDefault()
)

type sizeType int64
type subnetType int64

func (a *sizeType) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	var err error
	i, err := humanize.ParseBytes(string(text))
	if err != nil {
		return err
	}
	*a = sizeType(i)
	return nil
}

func (a *subnetType) UnmarshalText(text []byte) error {
	if len(text) < 1 {
		return nil
	}

	dec, err := netx.IPv4ToInt(string(text))
	if err != nil {
		return err
	}
	*a = subnetType(dec)
	return nil
}

// HealthCheckConfig contain healthcheck config
type HealthCheckConfig struct {
	Interval             int   `toml:"interval" default:"60"`
	Timeout              int   `toml:"timeout" default:"10"`
	CacheTTL             int64 `toml:"cache_ttl" default:"300"`
	EnableDefaultChecker bool  `toml:"enable_default_checker" default:"true"`
}

// Config contain all configs
type EruConfig struct {
	Enable   bool     `toml:"enable" default:"true"`
	Addrs    []string `toml:"addrs"`
	Username string   `toml:"username"`
	Password string   `toml:"password"`
	Podname  string   `toml:"podname" default:"virt"`

	Hostname          string   `toml:"-"`
	HeartbeatInterval int      `toml:"heartbeat_interval" default:"60"`
	Labels            []string `toml:"labels"` // node labels

	CheckOnlyMine bool `toml:"check_only_mine" default:"false"`

	HealthCheck HealthCheckConfig `toml:"healthcheck"`

	GlobalConnectionTimeout time.Duration `toml:"global_connection_timeout" default:"5s"`
}

// GetHealthCheckStatusTTL returns the TTL for health check status.
// Because selfmon is integrated in eru-core, so there returns 0.
func (config *EruConfig) GetHealthCheckStatusTTL() int64 {
	return 0
}

type HostConfig struct {
	ID      uint32     `json:"id" toml:"id"`
	Name    string     `json:"name" toml:"name"`
	Addr    string     `json:"addr" toml:"addr"`
	Type    string     `json:"type" toml:"type"`
	Subnet  subnetType `json:"subnet" toml:"subnet"`
	CPU     int        `json:"cpu" toml:"cpu"`
	Memory  sizeType   `json:"memory" toml:"memory"`
	Storage sizeType   `json:"storage" toml:"storage"`
}

type ETCDConfig struct {
	Prefix    string   `toml:"prefix" default:"/yavirt/v1"`
	Endpoints []string `toml:"endpoints" default:"[http://127.0.0.1:2379]"`
	Username  string   `toml:"username"`
	Password  string   `toml:"password"`
	CA        string   `toml:"ca"`
	Key       string   `toml:"key"`
	Cert      string   `toml:"cert"`
}

type CalicoConfig struct {
	ConfigFile   string   `toml:"config_file" default:"/etc/calico/calicoctl.cfg"`
	Nodename     string   `toml:"nodename"`
	PoolNames    []string `toml:"pools" default:"[clouddev]"`
	GatewayName  string   `toml:"gateway" default:"yavirt-cali-gw"`
	ETCDEnv      string   `toml:"etcd_env" default:"ETCD_ENDPOINTS"`
	IFNamePrefix string   `toml:"ifname_prefix" default:"cali"`
}

func (c *CalicoConfig) Check() error {
	if len(c.ConfigFile) == 0 && len(os.Getenv(c.ETCDEnv)) == 0 {
		return errors.New("either config_file or etcd_env must be set")
	}
	return nil
}

type CNIConfig struct {
	PluginPath   string `toml:"plugin_path" default:"/usr/bin/yavirt-cni"`
	ConfigPath   string `toml:"config_path" default:"/etc/cni/net.d/yavirt-cni.conf"`
	IFNamePrefix string `toml:"ifname_prefix" default:"yap"`
}

func (c *CNIConfig) Check() error {
	if c.PluginPath == "" || c.ConfigPath == "" {
		return errors.New("cni config must be set")
	}
	return nil
}

type VlanConfig struct {
	Subnet       subnetType `json:"subnet" toml:"subnet"`
	IFNamePrefix string     `toml:"ifname_prefix" default:"yap"`
}

func (c *VlanConfig) Check() error {
	return nil
}

type OVNConfig struct {
	NBAddrs      []string `toml:"nb_addrs" default:"[tcp:127.0.0.1:6641]"`
	OVSDBAddr    string   `toml:"ovsdb_addr" default:"unix:/var/run/openvswitch/db.sock"`
	IFNamePrefix string   `toml:"ifname_prefix" default:"yap"`
}

func (c *OVNConfig) Check() error {
	if len(c.NBAddrs) == 0 || c.OVSDBAddr == "" {
		return errors.New("ovn config must be set")
	}
	return nil
}

type NetworkConfig struct {
	Modes       []string     `toml:"modes" default:"[calico]"` // supported network modes
	DefaultMode string       `toml:"default_mode" default:"calico"`
	Calico      CalicoConfig `toml:"calico"`
	CNI         CNIConfig    `toml:"cni"`
	Vlan        VlanConfig   `toml:"vlan"`
	OVN         OVNConfig    `toml:"ovn"`
}

type CephConfig struct {
	MonitorAddrs []string `toml:"monitor_addrs"`
	Username     string   `toml:"username" default:"eru"`
	SecretUUID   string   `toml:"secret_uuid"`
}

type LocalConfig struct {
	Dir string `toml:"dir"`
}

type StorageConfig struct {
	InitGuestVolume bool        `toml:"init_guest_volume"`
	Ceph            CephConfig  `toml:"ceph"`
	Local           LocalConfig `toml:"local"`
}

type ResourceConfig struct {
	MinCPU          int      `toml:"min_cpu" default:"1"`
	MaxCPU          int      `toml:"max_cpu" default:"112"`
	MinMemory       int64    `toml:"min_memory" default:"536870912"`        // default: 512M
	MaxMemory       int64    `toml:"max_memory" default:"549755813888"`     // default: 512G
	ReservedMemory  int64    `toml:"reserved_memory" default:"10737418240"` // default: 10GB
	MinVolumeCap    int64    `toml:"min_volume" default:"1073741824"`
	MaxVolumeCap    int64    `toml:"max_volume" default:"1099511627776"`
	MaxVolumesCount int      `toml:"max_volumes_count" default:"8"`
	Bandwidth       int64    `toml:"bandwidth" default:"50000000000"`
	ExcludePCIs     []string `toml:"exclude_pcis"`

	GPUProductMap map[string]string `toml:"gpu_product_map"`
}

type VMAuthConfig struct {
	Username string `toml:"username" default:"root"`
	Password string `toml:"password" default:"root"`
}

type LogConfig struct {
	Level     string `toml:"level" default:"info"`
	UseJSON   bool   `toml:"use_json"`
	SentryDSN string `toml:"sentry_dsn"`
	Verbose   bool   `toml:"verbose"`
	// for file log
	Filename   string `toml:"filename"`
	MaxSize    int    `toml:"maxsize" default:"500"`
	MaxAge     int    `toml:"max_age" default:"28"`
	MaxBackups int    `toml:"max_backups" default:"3"`
}

// Config .
type Config struct {
	Env      string `toml:"env" default:"dev"`
	CertPath string `toml:"cert_path" default:"/etc/eru/tls"`

	MaxConcurrency         int      `toml:"max_concurrency" default:"100000"`
	ProfHTTPPort           int      `toml:"prof_http_port" default:"9999"`
	BindHTTPAddr           string   `toml:"bind_http_addr" default:"0.0.0.0:9696"`
	BindGRPCAddr           string   `toml:"bind_grpc_addr" default:"0.0.0.0:9697"`
	SkipGuestReportRegexps []string `toml:"skip_guest_report_regexps"`
	EnableLibvirtMetrics   bool     `toml:"enable_libvirt_metrics"`

	VirtTimeout        time.Duration `toml:"virt_timeout" default:"60m"`
	GracefulTimeout    time.Duration `toml:"graceful_timeout" default:"20s"`
	HealthCheckTimeout time.Duration `toml:"health_check_timeout" default:"2s"`
	QMPConnectTimeout  time.Duration `toml:"qmp_connect_timeout" default:"8s"`
	MemStatsPeriod     int           `toml:"mem_stats_period" default:"10"` // in seconds

	GADiskTimeout time.Duration `toml:"ga_disk_timeout" default:"16m"`
	GABootTimeout time.Duration `toml:"ga_boot_timeout" default:"30m"`

	ResizeVolumeMinRatio float64 `toml:"resize_volume_min_ratio" default:"0.001"`
	ResizeVolumeMinSize  int64   `toml:"resize_volume_min_size" default:"1073741824"` // default 1GB

	MaxSnapshotsCount     int `toml:"max_snapshots_count" default:"30"`
	SnapshotRestorableDay int `toml:"snapshot_restorable_days" default:"7"`

	MetaTimeout time.Duration `toml:"meta_timeout" default:"1m"`
	MetaType    string        `toml:"meta_type" default:"etcd"`

	VirtDir                 string `toml:"virt_dir" default:"/opt/yavirtd"`
	VirtFlockDir            string `toml:"virt_flock_dir"`
	VirtTmplDir             string `toml:"virt_temp_dir"`
	VirtCloudInitDir        string `toml:"virt_cloud_init_dir"`
	VirtBridge              string `toml:"virt_bridge" default:"yavirbr0"`
	VirtCPUCachePassthrough bool   `toml:"virt_cpu_cache_passthrough" default:"true"`

	Batches []*Batch `toml:"batches"`

	// system recovery
	RecoveryOn            bool          `toml:"recovery_on"`
	RecoveryMaxRetries    int           `toml:"recovery_max_retries" default:"2"`
	RecoveryRetryInterval time.Duration `toml:"recovery_retry_interval" default:"3m"`
	RecoveryInterval      time.Duration `toml:"recovery_interval" default:"10m"`

	// host-related config
	Host     HostConfig           `toml:"host"`
	Eru      EruConfig            `toml:"eru"`
	Etcd     ETCDConfig           `toml:"etcd"`
	Network  NetworkConfig        `toml:"network"`
	Storage  StorageConfig        `toml:"storage"`
	Resource ResourceConfig       `toml:"resource"`
	ImageHub vmitypes.Config      `toml:"image_hub"`
	Auth     coretypes.AuthConfig `toml:"auth"` // grpc auth
	VMAuth   VMAuthConfig         `toml:"vm_auth"`
	Log      LogConfig            `toml:"log"`
	Notify   bison.Config         `toml:"notify"`
}

func Hostname() string {
	return Conf.Host.Name
}

func newDefault() Config {
	conf := new(Config)
	defaults.SetDefaults(conf)
	return *conf
}

// Dump .
func (cfg *Config) Dump() (string, error) {
	return Encode(cfg)
}

// Load .
func (cfg *Config) Load(files ...string) error {
	for _, path := range files {
		if err := DecodeFile(path, cfg); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (cfg *Config) Prepare(c *cli.Context) (err error) {
	// try to initialize Hostname
	if c.String("addr") != "" {
		cfg.Host.Addr = c.String("addr")
	}
	if c.String("hostname") != "" {
		cfg.Host.Name = c.String("hostname")
	} else if cfg.Host.Name == "" {
		cfg.Host.Name, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if cfg.Host.Name == "" {
		cfg.Host.Name = strings.ReplaceAll(cfg.Host.Addr, ".", "-")
	}

	if c.String("log-level") != "" {
		cfg.Log.Level = c.String("log-level")
	}

	if len(c.StringSlice("core-addrs")) > 0 {
		cfg.Eru.Addrs = c.StringSlice("core-addrs")
	}
	if c.String("core-username") != "" {
		cfg.Eru.Username = c.String("core-username")
	}
	if c.String("core-password") != "" {
		cfg.Eru.Password = c.String("core-password")
	}
	// prepare ETCD_ENDPOINTS(Calico needs this environment variable)
	if !utils.FileExists(cfg.Network.Calico.ConfigFile) {
		cfg.Network.Calico.ConfigFile = ""
	}
	if cfg.Network.Calico.Nodename == "" {
		cfg.Network.Calico.Nodename = cfg.Host.Name
	}
	if len(cfg.Etcd.Endpoints) > 0 {
		if err = os.Setenv(cfg.Network.Calico.ETCDEnv, strings.Join(cfg.Etcd.Endpoints, ",")); err != nil {
			return err
		}
	}
	if cfg.CertPath != "" {
		if cfg.Etcd.CA == "" {
			cfg.Etcd.CA = filepath.Join(cfg.CertPath, "etcd", "ca.pem")
		}
		if cfg.Etcd.Cert == "" {
			cfg.Etcd.Cert = filepath.Join(cfg.CertPath, "etcd", "cert.pem")
		}
		if cfg.Etcd.Key == "" {
			cfg.Etcd.Key = filepath.Join(cfg.CertPath, "etcd", "key.pem")
		}
	}

	if cfg.Host.Addr == "" {
		return errors.New("Address must be provided")
	}
	// validate config values
	if cfg.Host.Name == "" {
		return errors.New("Hostname must be provided")
	}
	// Network
	if err := cfg.checkNetwork(); err != nil {
		return err
	}
	// eru
	if len(cfg.Eru.Addrs) == 0 {
		return errors.New("Core addresses are needed")
	}
	cfg.Eru.Hostname = cfg.Host.Name
	if err := cfg.ImageHub.CheckAndRefine(); err != nil {
		return err
	}
	return cfg.loadVirtDirs()
}

func (cfg *Config) checkNetwork() error {
	if len(cfg.Network.Modes) == 0 {
		return errors.New("Network modes must be provided")
	}
	if cfg.Network.DefaultMode == "" {
		cfg.Network.DefaultMode = cfg.Network.Modes[0]
	}
	var found bool
	for _, mode := range cfg.Network.Modes {
		if mode == cfg.Network.DefaultMode {
			found = true
			break
		}
		switch mode {
		case "cni":
			if err := cfg.Network.CNI.Check(); err != nil {
				return err
			}
		case "calico":
			if err := cfg.Network.Calico.Check(); err != nil {
				return err
			}
		case "ovn":
			if err := cfg.Network.OVN.Check(); err != nil {
				return err
			}
		case "vlan":
			if err := cfg.Network.Vlan.Check(); err != nil {
				return err
			}
		default:
			return errors.New("Invalid network mode")
		}
	}
	if !found {
		return errors.New("Invalid default network mode")
	}
	return nil
}

func (cfg *Config) loadVirtDirs() error {
	cfg.VirtFlockDir = filepath.Join(cfg.VirtDir, "flock")
	cfg.VirtTmplDir = filepath.Join(cfg.VirtDir, "template")
	cfg.VirtCloudInitDir = filepath.Join(cfg.VirtDir, "cloud-init")

	// ensure directories
	for _, d := range []string{cfg.VirtFlockDir, cfg.VirtTmplDir, cfg.VirtCloudInitDir} {
		if err := os.MkdirAll(d, 0755); err != nil && !os.IsExist(err) {
			return err
		}
		err := os.MkdirAll(d, 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}
