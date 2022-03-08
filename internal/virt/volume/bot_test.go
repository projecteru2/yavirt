package volume

import (
	"context"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/projecteru2/yavirt/pkg/meta"
	"github.com/projecteru2/yavirt/pkg/sh"
	shmocks "github.com/projecteru2/yavirt/pkg/sh/mocks"
	"github.com/projecteru2/yavirt/pkg/test/assert"
	"github.com/projecteru2/yavirt/pkg/test/mock"
	agentmocks "github.com/projecteru2/yavirt/internal/virt/agent/mocks"
	agenttypes "github.com/projecteru2/yavirt/internal/virt/agent/types"
	guestfstypes "github.com/projecteru2/yavirt/internal/virt/guestfs/types"
)

func TestResetFstabByUUID(t *testing.T) {
	vol, gfs := NewMockedVolume()
	gfs.On("GetFstabEntries").Return(map[string]string{
		"/dev/sda1": "/dev/sda1 / ext4 defaults 0 1",
		"UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5": "UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1",
		"LABEL=cloudimg-rootfs":                     "LABEL=cloudimg-rootfs / ext4 defaults 0 1",
	}, nil).Once()

	blkids := guestfstypes.Blkids{}
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda1", Label: "a", UUID: "34c4810a-3b02-11ec-a3d8-52540049d2ce"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda2", Label: "b", UUID: "a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda3", Label: "c", UUID: "581f0760-3b02-11ec-a3d8-52540049d2ce"})
	gfs.On("GetBlkids").Return(blkids, nil).Once()

	gfs.On("Write", "/etc/fstab", "UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1\n").Return(nil)
	assert.NilErr(t, vol.(*bot).resetFstab(gfs))
}
func TestResetFstabByDevname(t *testing.T) {
	vol, gfs := NewMockedVolume()
	gfs.On("GetFstabEntries").Return(map[string]string{
		"/dev/sda1": "/dev/sda1 / ext4 defaults 0 1",
		"UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5": "UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1",
		"LABEL=cloudimg-rootfs":                     "LABEL=cloudimg-rootfs / ext4 defaults 0 1",
	}, nil).Once()

	blkids := guestfstypes.Blkids{}
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/sda1", Label: "a", UUID: "34c4810a-3b02-11ec-a3d8-52540049d2ce"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/sda2", Label: "b", UUID: "99ad936c-3ba8-11ec-879b-52540049d2ce"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/sda3", Label: "c", UUID: "581f0760-3b02-11ec-a3d8-52540049d2ce"})
	gfs.On("GetBlkids").Return(blkids, nil).Once()

	gfs.On("Write", "/etc/fstab", "/dev/sda1 / ext4 defaults 0 1\n").Return(nil)
	assert.NilErr(t, vol.(*bot).resetFstab(gfs))
}

func TestResetFstabByLabel(t *testing.T) {
	vol, gfs := NewMockedVolume()
	gfs.On("GetFstabEntries").Return(map[string]string{
		"/dev/sda1": "/dev/sda1 / ext4 defaults 0 1",
		"UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5": "UUID=a3fe5809-c923-4f00-8a8c-5fe85ad9d1e5 / ext4 defaults 0 1",
		"LABEL=c": "LABEL=c / ext4 defaults 0 1",
	}, nil).Once()

	blkids := guestfstypes.Blkids{}
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda1", Label: "a", UUID: "34c4810a-3b02-11ec-a3d8-52540049d2ce"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda2", Label: "b", UUID: "99ad936c-3ba8-11ec-879b-52540049d2ce"})
	blkids.Add(&guestfstypes.Blkid{Dev: "/dev/vda3", Label: "c", UUID: "581f0760-3b02-11ec-a3d8-52540049d2ce"})
	gfs.On("GetBlkids").Return(blkids, nil).Once()

	gfs.On("Write", "/etc/fstab", "LABEL=c / ext4 defaults 0 1\n").Return(nil)
	assert.NilErr(t, vol.(*bot).resetFstab(gfs))
}

func TestConvertUserImage(t *testing.T) {
	shx := &shmocks.Shell{}
	defer shx.AssertExpectations(t)
	shx.On("Copy", mock.Anything, mock.Anything).Return(nil).Once()
	shx.On("Move", mock.Anything, mock.Anything).Return(nil).Once()

	cancel := sh.NewMockShell(shx)
	defer cancel()

	vol, gfs := NewMockedVolume()
	gfs.On("Distro").Return("centos", nil).Once()
	gfs.On("GetFstabEntries").Return(map[string]string{"/dev/sda1": "/dev/sda1 / ext4 defaults 0 1"}, nil).Once()
	gfs.On("GetBlkids").Return(guestfstypes.Blkids{}, nil).Once()
	gfs.On("Write", mock.Anything, mock.Anything).Return(nil).Once()
	gfs.On("Remove", mock.Anything).Return(nil).Twice()
	gfs.On("Close").Return(nil).Once()

	uimg, err := vol.ConvertUserImage("anrs", "aa")
	assert.NilErr(t, err)
	assert.NotNil(t, uimg)
	assert.Equal(t, "anrs", uimg.User)
	assert.Equal(t, "aa", uimg.Name)
	assert.Equal(t, "centos", uimg.Distro)
	assert.Equal(t, int64(0), uimg.Version)
	assert.Equal(t, "centos-anrs-aa-0.uimg", filepath.Base(uimg.Filepath()))
	assert.Equal(t, vol.(*bot).vol.Capacity, uimg.Size)
}

func TestStopSystemdServices(t *testing.T) {
	vol, _ := NewMockedVolume()

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	outFuser := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("/data: 125rw")),
		}
		done <- out
		return done
	}()

	outSystemctlStatus125 := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited: true,
			Code:   1,
			Base64Err: base64.StdEncoding.EncodeToString([]byte(
				"Failed to get unit for PID 12: PID 12 does not belong to any loaded unit.")),
		}
		done <- out
		return done
	}()

	outSystemctlStatus120 := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("X yavirt.service - A Service for Yavirt")),
		}
		done <- out
		return done
	}()

	outPpid := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("120")),
		}
		done <- out
		return done
	}()

	outSystemctlStop := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited: true,
			Code:   0,
		}
		done <- out
		return done
	}()

	outIsServiceYavirt := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("yavirt.service loaded active running")),
		}
		done <- out
		return done
	}()

	agent := &agentmocks.Interface{}
	defer agent.AssertExpectations(t)
	agent.On("ExecOutput", ctx, "fuser", "-m", "/data").Return(outFuser)
	agent.On("ExecOutput", ctx, "systemctl", "status", "125").Return(outSystemctlStatus125)
	agent.On("ExecOutput", ctx, "ps", "--ppid", "125").Return(outPpid)
	agent.On("ExecOutput", ctx, "systemctl", "status", "120").Return(outSystemctlStatus120)
	agent.On("ExecOutput", ctx, "systemctl", "list-units", "--all", "-t", "service",
		"--full", "--no-legend", "yavirt.service").Return(outIsServiceYavirt)
	agent.On("Exec", ctx, "systemctl", "stop", "yavirt.service").Return(outSystemctlStop)

	stoppedServices, err := vol.(*bot).stopSystemdServices(ctx, agent, "/data")
	assert.NilErr(t, err)
	assert.Equal(t, len(stoppedServices), 1)
}

func TestGetPpid(t *testing.T) {
	vol, _ := NewMockedVolume()

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	d := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("120")),
		}
		done <- out
		return done
	}()

	agent := &agentmocks.Interface{}
	defer agent.AssertExpectations(t)
	agent.On("ExecOutput", ctx, "ps", "--ppid", "123").Return(d)

	ppid, err := vol.(*bot).getPpid(ctx, agent, "123")
	assert.NilErr(t, err)
	assert.Equal(t, ppid, "120")
}

func TestGetServiceNameByPid(t *testing.T) {
	vol, _ := NewMockedVolume()

	ctx, cancel := meta.Context(context.Background())
	defer cancel()

	d := func() <-chan agenttypes.ExecStatus {
		var done = make(chan agenttypes.ExecStatus, 1)
		var out = agenttypes.ExecStatus{
			Exited:    true,
			Code:      0,
			Err:       nil,
			Base64Out: base64.StdEncoding.EncodeToString([]byte("@ yavirt.service - A Service for Yavirt")),
		}
		done <- out
		return done
	}()

	agent := &agentmocks.Interface{}
	defer agent.AssertExpectations(t)
	agent.On("ExecOutput", ctx, "systemctl", "status", "123").Return(d)

	name, se := vol.(*bot).getServiceNameByPid(ctx, agent, "123")
	assert.Equal(t, name, "yavirt.service")
	assert.Equal(t, se, "")
}
