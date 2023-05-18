package configs

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestHostConfig(t *testing.T) {
	ss := `
[host]
name = "host1"
subnet = "127.0.0.1"
cpu = 4
memory = "1gib"
storage = "40gi"
network = "calico"
	`
	cfg := Config{}
	_, err := toml.Decode(ss, &cfg)
	assert.Nil(t, err)
	assert.Equal(t, cfg.Host.Memory, sizeType(1*1024*1024*1024))
	assert.Equal(t, cfg.Host.Storage, sizeType(40*1024*1024*1024))
}
