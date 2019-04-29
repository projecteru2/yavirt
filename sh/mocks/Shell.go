// Code generated by mockery v2.3.0. DO NOT EDIT.

package mocks

import (
	context "context"
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// Shell is an autogenerated mock type for the Shell type
type Shell struct {
	mock.Mock
}

// Copy provides a mock function with given fields: src, dest
func (_m *Shell) Copy(src string, dest string) error {
	ret := _m.Called(src, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(src, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Exec provides a mock function with given fields: ctx, name, args
func (_m *Shell) Exec(ctx context.Context, name string, args ...string) error {
	_va := make([]interface{}, len(args))
	for _i := range args {
		_va[_i] = args[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, ...string) error); ok {
		r0 = rf(ctx, name, args...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ExecInOut provides a mock function with given fields: ctx, env, stdin, name, args
func (_m *Shell) ExecInOut(ctx context.Context, env map[string]string, stdin io.Reader, name string, args ...string) ([]byte, []byte, error) {
	_va := make([]interface{}, len(args))
	for _i := range args {
		_va[_i] = args[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, env, stdin, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(context.Context, map[string]string, io.Reader, string, ...string) []byte); ok {
		r0 = rf(ctx, env, stdin, name, args...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func(context.Context, map[string]string, io.Reader, string, ...string) []byte); ok {
		r1 = rf(ctx, env, stdin, name, args...)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(context.Context, map[string]string, io.Reader, string, ...string) error); ok {
		r2 = rf(ctx, env, stdin, name, args...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// Move provides a mock function with given fields: src, dest
func (_m *Shell) Move(src string, dest string) error {
	ret := _m.Called(src, dest)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(src, dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Remove provides a mock function with given fields: fpth
func (_m *Shell) Remove(fpth string) error {
	ret := _m.Called(fpth)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(fpth)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
