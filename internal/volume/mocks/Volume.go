// Code generated by mockery v2.42.0. DO NOT EDIT.

package mocks

import (
	agent "github.com/projecteru2/yavirt/internal/virt/agent"
	base "github.com/projecteru2/yavirt/internal/volume/base"

	context "context"

	guestfs "github.com/projecteru2/yavirt/internal/virt/guestfs"

	libvirt "github.com/projecteru2/yavirt/pkg/libvirt"

	mock "github.com/stretchr/testify/mock"

	types "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

// Volume is an autogenerated mock type for the Volume type
type Volume struct {
	mock.Mock
}

// AmplifyOffline provides a mock function with given fields: ctx, delta
func (_m *Volume) AmplifyOffline(ctx context.Context, delta int64) error {
	ret := _m.Called(ctx, delta)

	if len(ret) == 0 {
		panic("no return value specified for AmplifyOffline")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, delta)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AmplifyOnline provides a mock function with given fields: newCap, dom, ga
func (_m *Volume) AmplifyOnline(newCap int64, dom libvirt.Domain, ga agent.Interface) error {
	ret := _m.Called(newCap, dom, ga)

	if len(ret) == 0 {
		panic("no return value specified for AmplifyOnline")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(int64, libvirt.Domain, agent.Interface) error); ok {
		r0 = rf(newCap, dom, ga)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CaptureImage provides a mock function with given fields: imgName
func (_m *Volume) CaptureImage(imgName string) (*types.Image, error) {
	ret := _m.Called(imgName)

	if len(ret) == 0 {
		panic("no return value specified for CaptureImage")
	}

	var r0 *types.Image
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*types.Image, error)); ok {
		return rf(imgName)
	}
	if rf, ok := ret.Get(0).(func(string) *types.Image); ok {
		r0 = rf(imgName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Image)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(imgName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Check provides a mock function with given fields:
func (_m *Volume) Check() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Check")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Cleanup provides a mock function with given fields:
func (_m *Volume) Cleanup() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Cleanup")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Delete provides a mock function with given fields: force
func (_m *Volume) Delete(force bool) error {
	ret := _m.Called(force)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GenerateID provides a mock function with given fields:
func (_m *Volume) GenerateID() {
	_m.Called()
}

// GenerateXML provides a mock function with given fields:
func (_m *Volume) GenerateXML() ([]byte, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GenerateXML")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]byte, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCreatedTime provides a mock function with given fields:
func (_m *Volume) GetCreatedTime() int64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetCreatedTime")
	}

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// GetDevice provides a mock function with given fields:
func (_m *Volume) GetDevice() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDevice")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetGfx provides a mock function with given fields:
func (_m *Volume) GetGfx() (guestfs.Guestfs, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetGfx")
	}

	var r0 guestfs.Guestfs
	var r1 error
	if rf, ok := ret.Get(0).(func() (guestfs.Guestfs, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() guestfs.Guestfs); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(guestfs.Guestfs)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGuestID provides a mock function with given fields:
func (_m *Volume) GetGuestID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetGuestID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetHostname provides a mock function with given fields:
func (_m *Volume) GetHostname() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetHostname")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetID provides a mock function with given fields:
func (_m *Volume) GetID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetMountDir provides a mock function with given fields:
func (_m *Volume) GetMountDir() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetMountDir")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetSize provides a mock function with given fields:
func (_m *Volume) GetSize() int64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSize")
	}

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// GetStatus provides a mock function with given fields:
func (_m *Volume) GetStatus() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetStatus")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetVer provides a mock function with given fields:
func (_m *Volume) GetVer() int64 {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetVer")
	}

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// GetXMLQStr provides a mock function with given fields:
func (_m *Volume) GetXMLQStr() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetXMLQStr")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IncrVer provides a mock function with given fields:
func (_m *Volume) IncrVer() {
	_m.Called()
}

// IsSys provides a mock function with given fields:
func (_m *Volume) IsSys() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsSys")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Lock provides a mock function with given fields:
func (_m *Volume) Lock() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Lock")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MetaKey provides a mock function with given fields:
func (_m *Volume) MetaKey() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for MetaKey")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Mount provides a mock function with given fields: ctx, ga
func (_m *Volume) Mount(ctx context.Context, ga agent.Interface) error {
	ret := _m.Called(ctx, ga)

	if len(ret) == 0 {
		panic("no return value specified for Mount")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, agent.Interface) error); ok {
		r0 = rf(ctx, ga)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Name provides a mock function with given fields:
func (_m *Volume) Name() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Name")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewSnapshotAPI provides a mock function with given fields:
func (_m *Volume) NewSnapshotAPI() base.SnapshotAPI {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for NewSnapshotAPI")
	}

	var r0 base.SnapshotAPI
	if rf, ok := ret.Get(0).(func() base.SnapshotAPI); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(base.SnapshotAPI)
		}
	}

	return r0
}

// PrepareDataDisk provides a mock function with given fields: _a0
func (_m *Volume) PrepareDataDisk(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for PrepareDataDisk")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PrepareSysDisk provides a mock function with given fields: _a0, _a1, _a2
func (_m *Volume) PrepareSysDisk(_a0 context.Context, _a1 *types.Image, _a2 ...base.Option) error {
	_va := make([]interface{}, len(_a2))
	for _i := range _a2 {
		_va[_i] = _a2[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _a0, _a1)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PrepareSysDisk")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *types.Image, ...base.Option) error); ok {
		r0 = rf(_a0, _a1, _a2...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Repair provides a mock function with given fields:
func (_m *Volume) Repair() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Repair")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Save provides a mock function with given fields:
func (_m *Volume) Save() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Save")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetDevice provides a mock function with given fields: dev
func (_m *Volume) SetDevice(dev string) {
	_m.Called(dev)
}

// SetGuestID provides a mock function with given fields: id
func (_m *Volume) SetGuestID(id string) {
	_m.Called(id)
}

// SetHostname provides a mock function with given fields: name
func (_m *Volume) SetHostname(name string) {
	_m.Called(name)
}

// SetSize provides a mock function with given fields: size
func (_m *Volume) SetSize(size int64) {
	_m.Called(size)
}

// SetStatus provides a mock function with given fields: st, force
func (_m *Volume) SetStatus(st string, force bool) error {
	ret := _m.Called(st, force)

	if len(ret) == 0 {
		panic("no return value specified for SetStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(st, force)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetVer provides a mock function with given fields: _a0
func (_m *Volume) SetVer(_a0 int64) {
	_m.Called(_a0)
}

// Umount provides a mock function with given fields: ctx, ga
func (_m *Volume) Umount(ctx context.Context, ga agent.Interface) error {
	ret := _m.Called(ctx, ga)

	if len(ret) == 0 {
		panic("no return value specified for Umount")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, agent.Interface) error); ok {
		r0 = rf(ctx, ga)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Unlock provides a mock function with given fields:
func (_m *Volume) Unlock() {
	_m.Called()
}

// NewVolume creates a new instance of Volume. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewVolume(t interface {
	mock.TestingT
	Cleanup(func())
}) *Volume {
	mock := &Volume{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
