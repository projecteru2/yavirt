package ovn

import (
	"testing"

	. "github.com/agiledragon/gomonkey/v2"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/stretchr/testify/assert"
)

func TestCreateEndpointNetwork(t *testing.T) {
	inputArgs := types.EndpointArgs{
		GuestID: "00052017027003203659270000000003",
		MAC:     "00:55:00:00:00:03",
		OVN: types.OVNArgs{
			LogicalSwitchUUID: "haha-kaka",
		},
	}
	lspUUID := "kaka-123456"
	cfg := &configs.OVNConfig{
		NBAddrs: []string{
			"tcp:127.0.0.1:6641",
		},
	}
	d, err := NewDriver(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, d)
	patches := ApplyPrivateMethod(d, "createLogicalSwitchPort", func(_ *Driver, args *types.EndpointArgs) (string, error) {
		assert.Equal(t, inputArgs.OVN.LogicalSwitchUUID, args.OVN.LogicalSwitchUUID)
		return lspUUID, nil
	})
	defer patches.Reset()

	ls := &LogicalSwitch{
		UUID: "haha-kaka",
		Name: "test-ls",
		Config: map[string]string{
			"subnet": "192.168.110.1/24",
		},
	}
	patches = ApplyPrivateMethod(d, "getLogicalSwitch", func(_ *Driver, uuid string) (*LogicalSwitch, error) {
		assert.Equal(t, inputArgs.OVN.LogicalSwitchUUID, uuid)
		return ls, nil
	})
	defer patches.Reset()

	addr := "00:11:22:33:44:55 192.168.110.3"
	lsp := &LogicalSwitchPort{
		UUID: "haha-kaka",
		Name: "test-lsp",
		Addresses: []string{
			"00:11:22:33:44:55 dynamic",
		},
		DynamicAddresses: &addr,
	}
	patches = ApplyPrivateMethod(d, "getLogicalSwitchPort", func(d *Driver, uuid string) (*LogicalSwitchPort, error) {
		assert.Equal(t, lspUUID, uuid)
		return lsp, nil
	})
	defer patches.Reset()
	args, rollback, err := d.CreateEndpointNetwork(inputArgs)
	assert.Nil(t, err)
	assert.Nil(t, rollback)
	assert.Len(t, args.IPs, 1)
	assert.Equal(t, args.IPs[0].IPAddr(), "192.168.110.3")
}
