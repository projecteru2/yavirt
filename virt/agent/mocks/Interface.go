// Code generated by mockery 2.9.4. DO NOT EDIT.

package mocks

import (
	context "context"

	config "github.com/projecteru2/yavirt/config"

	mock "github.com/stretchr/testify/mock"

	types "github.com/projecteru2/yavirt/virt/agent/types"
)

// Interface is an autogenerated mock type for the Interface type
type Interface struct {
	mock.Mock
}

// AppendLine provides a mock function with given fields: filepath, p
func (_m *Interface) AppendLine(filepath string, p []byte) error {
	ret := _m.Called(filepath, p)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []byte) error); ok {
		r0 = rf(filepath, p)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Blkid provides a mock function with given fields: ctx, dev
func (_m *Interface) Blkid(ctx context.Context, dev string) (string, error) {
	ret := _m.Called(ctx, dev)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string) string); ok {
		r0 = rf(ctx, dev)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, dev)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Close provides a mock function with given fields:
func (_m *Interface) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CloseFile provides a mock function with given fields: handle
func (_m *Interface) CloseFile(handle int) error {
	ret := _m.Called(handle)

	var r0 error
	if rf, ok := ret.Get(0).(func(int) error); ok {
		r0 = rf(handle)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Exec provides a mock function with given fields: ctx, prog, args
func (_m *Interface) Exec(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus {
	_va := make([]interface{}, len(args))
	for _i := range args {
		_va[_i] = args[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, prog)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 <-chan types.ExecStatus
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) <-chan types.ExecStatus); ok {
		r0 = rf(ctx, prog, args...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan types.ExecStatus)
		}
	}

	return r0
}

// ExecBatch provides a mock function with given fields: bat
func (_m *Interface) ExecBatch(bat *config.Batch) error {
	ret := _m.Called(bat)

	var r0 error
	if rf, ok := ret.Get(0).(func(*config.Batch) error); ok {
		r0 = rf(bat)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ExecOutput provides a mock function with given fields: ctx, prog, args
func (_m *Interface) ExecOutput(ctx context.Context, prog string, args ...string) <-chan types.ExecStatus {
	_va := make([]interface{}, len(args))
	for _i := range args {
		_va[_i] = args[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, prog)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 <-chan types.ExecStatus
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) <-chan types.ExecStatus); ok {
		r0 = rf(ctx, prog, args...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan types.ExecStatus)
		}
	}

	return r0
}

// FlushFile provides a mock function with given fields: handle
func (_m *Interface) FlushFile(handle int) error {
	ret := _m.Called(handle)

	var r0 error
	if rf, ok := ret.Get(0).(func(int) error); ok {
		r0 = rf(handle)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDiskfree provides a mock function with given fields: ctx, mnt
func (_m *Interface) GetDiskfree(ctx context.Context, mnt string) (*types.Diskfree, error) {
	ret := _m.Called(ctx, mnt)

	var r0 *types.Diskfree
	if rf, ok := ret.Get(0).(func(context.Context, string) *types.Diskfree); ok {
		r0 = rf(ctx, mnt)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Diskfree)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, mnt)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Grep provides a mock function with given fields: ctx, keyword, filepath
func (_m *Interface) Grep(ctx context.Context, keyword string, filepath string) (bool, error) {
	ret := _m.Called(ctx, keyword, filepath)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(ctx, keyword, filepath)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, keyword, filepath)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsFile provides a mock function with given fields: ctx, filepath
func (_m *Interface) IsFile(ctx context.Context, filepath string) (bool, error) {
	ret := _m.Called(ctx, filepath)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, filepath)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, filepath)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsFolder provides a mock function with given fields: ctx, path
func (_m *Interface) IsFolder(ctx context.Context, path string) (bool, error) {
	ret := _m.Called(ctx, path)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, path)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OpenFile provides a mock function with given fields: path, mode
func (_m *Interface) OpenFile(path string, mode string) (int, error) {
	ret := _m.Called(path, mode)

	var r0 int
	if rf, ok := ret.Get(0).(func(string, string) int); ok {
		r0 = rf(path, mode)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(path, mode)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Ping provides a mock function with given fields: ctx
func (_m *Interface) Ping(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReadFile provides a mock function with given fields: handle, p
func (_m *Interface) ReadFile(handle int, p []byte) (int, bool, error) {
	ret := _m.Called(handle, p)

	var r0 int
	if rf, ok := ret.Get(0).(func(int, []byte) int); ok {
		r0 = rf(handle, p)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(int, []byte) bool); ok {
		r1 = rf(handle, p)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, []byte) error); ok {
		r2 = rf(handle, p)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// RemoveAll provides a mock function with given fields: ctx, path
func (_m *Interface) RemoveAll(ctx context.Context, path string) error {
	ret := _m.Called(ctx, path)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SeekFile provides a mock function with given fields: handle, offset, whence
func (_m *Interface) SeekFile(handle int, offset int, whence int) (int, bool, error) {
	ret := _m.Called(handle, offset, whence)

	var r0 int
	if rf, ok := ret.Get(0).(func(int, int, int) int); ok {
		r0 = rf(handle, offset, whence)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(int, int, int) bool); ok {
		r1 = rf(handle, offset, whence)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int, int) error); ok {
		r2 = rf(handle, offset, whence)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Touch provides a mock function with given fields: ctx, filepath
func (_m *Interface) Touch(ctx context.Context, filepath string) error {
	ret := _m.Called(ctx, filepath)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, filepath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WriteFile provides a mock function with given fields: handle, buf
func (_m *Interface) WriteFile(handle int, buf []byte) error {
	ret := _m.Called(handle, buf)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, []byte) error); ok {
		r0 = rf(handle, buf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
