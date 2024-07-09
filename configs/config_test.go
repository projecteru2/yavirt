package configs

import (
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestHostConfig(t *testing.T) {
	ss := `
meta_timeout = "3m"
ga_disk_timeout = "6m"
ga_boot_timeout = "10m"

[host]
name = "host1"
subnet = "127.0.0.1"
cpu = 4
memory = "1gib"
storage = "40gi"
network = "calico"

[resource.gpu_product_map]
"Nvidia 3070" = "nvidia-3070"
"Nvidia 4090" = "nvidia-4090"
	`
	cfg := Config{}
	_, err := toml.Decode(ss, &cfg)
	assert.Nil(t, err)
	assert.Equal(t, cfg.Host.Subnet, subnetType(2130706433))
	assert.Equal(t, cfg.Host.Memory, sizeType(1*1024*1024*1024))
	assert.Equal(t, cfg.Host.Storage, sizeType(40*1024*1024*1024))
	assert.Equal(t, cfg.MetaTimeout, 3*time.Minute)
	assert.Equal(t, cfg.GADiskTimeout, 6*time.Minute)
	assert.Equal(t, cfg.GABootTimeout, 10*time.Minute)
	assert.Equal(t, cfg.Resource.GPUProductMap, map[string]string{
		"Nvidia 3070": "nvidia-3070",
		"Nvidia 4090": "nvidia-4090",
	})

	ss = `
subnet = ""
memory = ""	
storage = 0
	`
	host := HostConfig{}
	_, err = toml.Decode(ss, &host)
	assert.Nil(t, err)
	assert.Equal(t, host.Memory, sizeType(0))
	assert.Equal(t, host.Storage, sizeType(0))
	assert.Equal(t, host.Subnet, subnetType(0))

	ss = `
memory = 1234
	`
	host = HostConfig{}
	_, err = toml.Decode(ss, &host)
	assert.Nil(t, err)
	assert.Equal(t, host.Memory, sizeType(1234))
}

func TestDefault(t *testing.T) {
	cfg := newDefault()
	assert.Equal(t, cfg.BindHTTPAddr, "0.0.0.0:9696")
	assert.Equal(t, cfg.BindGRPCAddr, "0.0.0.0:9697")
	assert.Equal(t, cfg.Resource.MinCPU, 1)
	assert.Equal(t, cfg.Resource.MaxCPU, 112)
	assert.Equal(t, cfg.MemStatsPeriod, 10)
	assert.Equal(t, cfg.ImageHub.Type, "docker")
	assert.Equal(t, cfg.ImageHub.Docker.Endpoint, "unix:///var/run/docker.sock")
	// assert.Equal(t, cfg.ImageHub.PullPolicy, "always")

	assert.Equal(t, cfg.VirtDir, "/opt/yavirtd")
	assert.Equal(t, cfg.VirtBridge, "yavirbr0")
	// eru
	assert.Equal(t, cfg.Eru.Enable, true)
	assert.Equal(t, cfg.Eru.GlobalConnectionTimeout, 5*time.Second)
	assert.Equal(t, cfg.Eru.HeartbeatInterval, 60)
	assert.Equal(t, cfg.Eru.HealthCheck.Interval, 60)
	assert.Equal(t, cfg.Eru.HealthCheck.Timeout, 10)
	assert.Equal(t, cfg.Eru.HealthCheck.CacheTTL, int64(300))
	// etcd
	assert.Equal(t, cfg.Etcd.Endpoints, []string{"http://127.0.0.1:2379"})
	assert.Equal(t, cfg.Etcd.Prefix, "/yavirt/v1")

	assert.Equal(t, cfg.MetaTimeout, time.Minute)
	assert.Equal(t, cfg.MetaType, "etcd")

	assert.Equal(t, cfg.GADiskTimeout, 16*time.Minute)
	assert.Equal(t, cfg.GABootTimeout, 30*time.Minute)

	assert.Equal(t, cfg.GracefulTimeout, 20*time.Second)
	assert.Equal(t, cfg.VirtTimeout, time.Hour)
	assert.Equal(t, cfg.HealthCheckTimeout, 2*time.Second)
	assert.Equal(t, cfg.QMPConnectTimeout, 8*time.Second)

	assert.Equal(t, cfg.ResizeVolumeMinRatio, 0.001)
	assert.Equal(t, cfg.ResizeVolumeMinSize, int64(1073741824))

	assert.Equal(t, cfg.MaxConcurrency, 100000)
	assert.Equal(t, cfg.MaxSnapshotsCount, 30)
	assert.Equal(t, cfg.SnapshotRestorableDay, 7)

	assert.False(t, cfg.RecoveryOn)
	assert.Equal(t, cfg.RecoveryMaxRetries, 2)
	assert.Equal(t, cfg.RecoveryRetryInterval, 3*time.Minute)
	assert.Equal(t, cfg.Network.OVN.NBAddrs, []string{"tcp:127.0.0.1:6641"})
}
