package ovn

import (
	"fmt"
	"testing"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/network/types"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestLS(t *testing.T) {
	cfg := &configs.OVNConfig{
		NBAddrs: []string{
			"tcp:192.168.160.25:6641",
		},
	}
	d, err := NewDriver(cfg)
	assert.Nil(t, err)
	// err = d.createLogicalSwitch("ut-test", "192.168.111.0/24")
	// err = d.createLogicalSwitch("ut-test", "")
	// assert.Nil(t, err)
	newLS, err := d.getLogicalSwitch("c343cbde-67b4-4b86-9cc1-9452b270f6f6")
	assert.Nil(t, err)
	fmt.Printf("+++++++ new ls: %v\n", newLS)
}

func TestLSP(t *testing.T) {
	cfg := &configs.OVNConfig{
		NBAddrs: []string{
			"tcp:192.168.160.25:6641",
		},
	}
	d, err := NewDriver(cfg)
	assert.Nil(t, err)
	lsUUID, err := d.createLogicalSwitch("ut-test", "192.168.111.0/24")
	assert.Nil(t, err)
	// defer d.deleteLogicalSwitch("ut-test")
	// lsUUID := "8c896d59-20d7-4b5f-9690-5b75ae03f787"
	args := &types.EndpointArgs{
		GuestID: "00052017027003203659270000000003",
		MAC:     "00:55:00:00:00:03",
		OVN: types.OVNArgs{
			LogicalSwitchUUID: lsUUID,
		},
	}
	uuid, err := d.createLogicalSwitchPort(args)
	assert.Nil(t, err)
	fmt.Printf("++++++++ uuid: %s\n", uuid)
	// time.Sleep(time.Millisecond)
	// d.nbCli = nil
	lsp, err := d.getLogicalSwitchPort(uuid)
	assert.Nil(t, err)
	fmt.Printf("++++++++ lsp: %v\n", lsp)
	assert.NotNil(t, lsp.DynamicAddresses)
	lsp1, err := d.getLogicalSwitchPortByName(LSPName(args.GuestID))
	assert.Nil(t, err)
	fmt.Printf("++++++++ lsp1: %v\n", lsp1)
}

func TestDeleteLSP(t *testing.T) {
	// lsUUID := "46333352-cfbb-45e2-a172-c0076b985718"
	guestID := "00009017034928197341160000000001"
	cfg := &configs.OVNConfig{
		NBAddrs: []string{
			"tcp:192.168.160.25:6641",
		},
	}
	d, err := NewDriver(cfg)
	assert.Nil(t, err)
	ls, err := d.getLogicalSwitchPortByName(LSPName(guestID))
	assert.Nil(t, err)
	fmt.Printf("+++++++++++ %v\n", ls)
	err = d.deleteLogicalSwitchPort(&types.EndpointArgs{
		GuestID: guestID,
		// OVNLogicSwitchUUID: lsUUID,
		OVN: types.OVNArgs{
			LogicalSwitchName: "ut-test",
		},
	})
	assert.Nil(t, err)
}

func TestSetExternalIDs(t *testing.T) {
	cfg := &configs.OVNConfig{
		NBAddrs: []string{
			"tcp:192.168.160.25:6641",
		},
		OVSDBAddr: "tcp:192.168.160.26:6640",
	}
	d, err := NewDriver(cfg)
	assert.Nil(t, err)
	err = d.setExternalID("vm2", "iface-id", "vm2222222")
	assert.Nil(t, err)
}
