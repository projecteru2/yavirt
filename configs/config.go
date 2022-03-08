package configs

import (
	"crypto/tls"
	"path/filepath"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/pkg/transport"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// DefaultTemplate .
const DefaultTemplate = `
env = "dev"
prof_http_port = 9999
bind_http_addr = "0.0.0.0:9696"
bind_grpc_addr = "0.0.0.0:9697"
graceful_timeout = "20s"
virt_timeout = "1h"
health_check_timeout = "2s"
qmp_connect_timeout = "8s"
cni_plugin_path = "/usr/bin/yavirt-cni"
cni_config_path = "/etc/cni/net.d/yavirt-cni.conf"

resize_volume_min_ratio = 0.05
resize_volume_min_size = 10737418240

min_cpu = 1
max_cpu = 64
min_memory = 1073741824
max_memory = 68719476736
min_volume = 1073741824
max_volume = 1099511627776
max_volumes_count = 8
max_snapshots_count = 30
snapshot_restorable_days = 7

meta_timeout = "1m"
meta_type = "etcd"

virt_dir = "/tmp/virt"
virt_bridge = "virbr0"
virt_cpu_cache_passthrough = true

calico_gateway = "yavirt-cali-gw"
calico_pools = ["clouddev"]
calico_etcd_env = "ETCD_ENDPOINTS"

log_level = "info"

etcd_prefix = "/yavirt-dev/v1"
etcd_endpoints = ["http://127.0.0.1:2379"]

core_addr = "127.0.0.1:5001"
core_username = "admin"
core_password = "password"
core_status_check_interval = "64s"
core_nodestatus_ttl = "16m"

ga_disk_timeout = "16m"
ga_boot_timeout = "30m"

recovery_on = false
recovery_max_retries = 2
recovery_retry_interval = "3m"
recovery_interval = "10m"
`

// Conf .
var Conf = newDefault()

// Config .
type Config struct {
	Env                    string   `toml:"env"`
	ProfHTTPPort           int      `toml:"prof_http_port"`
	BindHTTPAddr           string   `toml:"bind_http_addr"`
	BindGRPCAddr           string   `toml:"bind_grpc_addr"`
	SkipGuestReportRegexps []string `toml:"skip_guest_report_regexps"`
	EnabledCalicoCNI       bool     `toml:"enabled_calico_cni"`
	CNIPluginPath          string   `toml:"cni_plugin_path"`
	CNIConfigPath          string   `toml:"cni_config_path"`

	VirtTimeout        Duration `toml:"virt_timeout"`
	GracefulTimeout    Duration `toml:"graceful_timeout"`
	HealthCheckTimeout Duration `toml:"health_check_timeout"`
	QMPConnectTimeout  Duration `toml:"qmp_connect_timeout"`

	ImageHubDomain    string `toml:"image_hub_domain"`
	ImageHubNamespace string `toml:"image_hub_namespace"`

	GADiskTimeout Duration `toml:"ga_disk_timeout"`
	GABootTimeout Duration `toml:"ga_boot_timeout"`

	ResizeVolumeMinRatio float64 `toml:"resize_volume_min_ratio"`
	ResizeVolumeMinSize  int64   `toml:"resize_volume_min_size"`

	MinCPU                int   `toml:"min_cpu"`
	MaxCPU                int   `toml:"max_cpu"`
	MinMemory             int64 `toml:"min_memory"`
	MaxMemory             int64 `toml:"max_memory"`
	MinVolumeCap          int64 `toml:"min_volume"`
	MaxVolumeCap          int64 `toml:"max_volume"`
	MaxVolumesCount       int   `toml:"max_volumes_count"`
	MaxSnapshotsCount     int   `toml:"max_snapshots_count"`
	SnapshotRestorableDay int   `toml:"snapshot_restorable_days"`

	CalicoConfigFile  string   `toml:"calico_config_file"`
	CalicoPoolNames   []string `toml:"calico_pools"`
	CalicoGatewayName string   `toml:"calico_gateway"`
	CalicoETCDEnv     string   `toml:"calico_etcd_env"`

	MetaTimeout Duration `toml:"meta_timeout"`
	MetaType    string   `toml:"meta_type"`

	VirtDir                 string `toml:"virt_dir"`
	VirtFlockDir            string `toml:"virt_flock_dir"`
	VirtTmplDir             string `toml:"virt_temp_dir"`
	VirtSockDir             string `toml:"virt_sock_dir"`
	VirtBridge              string `toml:"virt_bridge"`
	VirtCPUCachePassthrough bool   `toml:"virt_cpu_cache_passthrough"`

	LogLevel  string `toml:"log_level"`
	LogFile   string `toml:"log_file"`
	LogSentry string `toml:"log_sentry"`

	EtcdPrefix    string   `toml:"etcd_prefix"`
	EtcdEndpoints []string `toml:"etcd_endpoints"`
	EtcdUsername  string   `toml:"etcd_username"`
	EtcdPassword  string   `toml:"etcd_password"`
	EtcdCA        string   `toml:"etcd_ca"`
	EtcdKey       string   `toml:"etcd_key"`
	EtcdCert      string   `toml:"etcd_cert"`

	CoreAddr                string   `toml:"core_addr"`
	CoreUsername            string   `toml:"core_username"`
	CorePassword            string   `toml:"core_password"`
	CoreStatusCheckInterval Duration `toml:"core_status_check_interval"`
	CoreNodeStatusTTL       Duration `toml:"core_nodestatus_ttl"`
	CoreNodename            string   `toml:"core_nodename"`

	Batches []*Batch `toml:"batches"`

	// system recovery
	RecoveryOn            bool     `toml:"recovery_on"`
	RecoveryMaxRetries    int      `toml:"recovery_max_retries"`
	RecoveryRetryInterval Duration `toml:"recovery_retry_interval"`
	RecoveryInterval      Duration `toml:"recovery_interval"`
}

func newDefault() Config {
	var conf Config
	if err := Decode(DefaultTemplate, &conf); err != nil {
		log.FatalStack(err)
	}

	conf.loadVirtDirs()

	return conf
}

// Dump .
func (c *Config) Dump() (string, error) {
	return Encode(c)
}

// Load .
func (c *Config) Load(files []string) error {
	for _, path := range files {
		if err := c.load(path); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *Config) load(file string) error {
	if err := DecodeFile(file, c); err != nil {
		return errors.Trace(err)
	}

	c.loadVirtDirs()

	return nil
}

func (c *Config) loadVirtDirs() {
	c.VirtFlockDir = filepath.Join(c.VirtDir, "flock")
	c.VirtTmplDir = filepath.Join(c.VirtDir, "template")
	c.VirtSockDir = filepath.Join(c.VirtDir, "sock")
}

// NewEtcdConfig .
func (c *Config) NewEtcdConfig() (etcdcnf clientv3.Config, err error) {
	etcdcnf.Endpoints = c.EtcdEndpoints
	etcdcnf.Username = c.EtcdUsername
	etcdcnf.Password = c.EtcdPassword
	etcdcnf.TLS, err = c.newEtcdTLSConfig()
	return
}

func (c *Config) newEtcdTLSConfig() (*tls.Config, error) {
	if len(c.EtcdCA) < 1 || len(c.EtcdKey) < 1 || len(c.EtcdCert) < 1 {
		return nil, nil
	}

	return transport.TLSInfo{
		TrustedCAFile: c.EtcdCA,
		KeyFile:       c.EtcdKey,
		CertFile:      c.EtcdCert,
	}.ClientConfig()
}

// CoreGuestStatusTTL .
func (c *Config) CoreGuestStatusTTL() time.Duration {
	return 3 * c.CoreStatusCheckInterval.Duration() //nolint:gomnd // TTL is 3 times the interval
}

// CoreGuestStatusCheckInterval .
func (c *Config) CoreGuestStatusCheckInterval() time.Duration {
	return c.CoreStatusCheckInterval.Duration()
}

// CoreGRPCTimeout .
func (c *Config) CoreGRPCTimeout() time.Duration {
	return c.CoreStatusReportInterval() / 3 //nolint:gomnd // report timeout 3 times per interval
}

// CoreStatusReportInterval .
func (c *Config) CoreStatusReportInterval() time.Duration {
	return c.CoreStatusCheckInterval.Duration() / 3 //nolint:gomnd // report 3 times every check
}

// HasImageHub indicates whether the config has ImageHub configurations.
func (c *Config) HasImageHub() bool {
	return len(c.ImageHubDomain) > 0 && len(c.ImageHubNamespace) > 0
}
