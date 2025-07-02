package types

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/projecteru2/yavirt/pkg/test/assert"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

func TestGenContent(t *testing.T) {
	cfg := &CloudInitConfig{
		Username: "root",
		Password: "passwd",
		OS: &vmitypes.OSInfo{
			Type: "linux",
		},
	}
	user, _, _, err := cfg.GenFilesContent()
	assert.Nil(t, err)
	// it is ugly here, but using yaml lib is too complex here
	assert.True(t, strings.Contains(user, "name: \"root\""))
	assert.True(t, strings.Contains(user, "plain_text_passwd: \"passwd\""))
	assert.False(t, strings.Contains(user, "- echo hello"))
	assert.False(t, strings.Contains(user, "- path: foo"))
	assert.False(t, strings.Contains(user, "content: "))

	cfg.Commands = []string{"echo hello"}
	cfg.Files = map[string][]byte{
		"foo": []byte("bar\nbar1"),
	}
	user, _, _, err = cfg.GenFilesContent()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(user, "name: \"root\""))
	assert.True(t, strings.Contains(user, "plain_text_passwd: \"passwd\""))
	assert.True(t, strings.Contains(user, "- echo hello"))
	assert.True(t, strings.Contains(user, "- path: foo"))
	assert.True(t, strings.Contains(user, fmt.Sprintf("content: %s", base64.StdEncoding.EncodeToString([]byte("bar\nbar1")))))
}

func TestNetwork(t *testing.T) {
	cfg := &CloudInitConfig{
		CIDR: "10.10.10.1/24",
		DefaultGW: CloudInitGateway{
			IP:     "10.10.10.111",
			OnLink: true,
		},
		OS: &vmitypes.OSInfo{
			Type: "linux",
		},
	}
	_, _, network, err := cfg.GenFilesContent()
	assert.Nil(t, err)
	// fmt.Printf("%s\n", network)
	assert.True(t, strings.Contains(network, "10.10.10.1/24"))
	assert.True(t, strings.Contains(network, "via: 10.10.10.111"))
	assert.True(t, strings.Contains(network, "on-link: true"))
}
