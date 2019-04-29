// Code generated by mockery 2.9.4. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// File is an autogenerated mock type for the File type
type File struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *File) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Flush provides a mock function with given fields:
func (_m *File) Flush() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Open provides a mock function with given fields:
func (_m *File) Open() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Read provides a mock function with given fields: p
func (_m *File) Read(p []byte) (int, error) {
	ret := _m.Called(p)

	var r0 int
	if rf, ok := ret.Get(0).(func([]byte) int); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadAt provides a mock function with given fields: dest, pos
func (_m *File) ReadAt(dest []byte, pos int) (int, error) {
	ret := _m.Called(dest, pos)

	var r0 int
	if rf, ok := ret.Get(0).(func([]byte, int) int); ok {
		r0 = rf(dest, pos)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte, int) error); ok {
		r1 = rf(dest, pos)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Seek provides a mock function with given fields: offset, whence
func (_m *File) Seek(offset int, whence int) (int, error) {
	ret := _m.Called(offset, whence)

	var r0 int
	if rf, ok := ret.Get(0).(func(int, int) int); ok {
		r0 = rf(offset, whence)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(offset, whence)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Tail provides a mock function with given fields: n
func (_m *File) Tail(n int) ([]byte, error) {
	ret := _m.Called(n)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(int) []byte); ok {
		r0 = rf(n)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(n)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Write provides a mock function with given fields: p
func (_m *File) Write(p []byte) (int, error) {
	ret := _m.Called(p)

	var r0 int
	if rf, ok := ret.Get(0).(func([]byte) int); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WriteLine provides a mock function with given fields: p
func (_m *File) WriteLine(p []byte) (int, error) {
	ret := _m.Called(p)

	var r0 int
	if rf, ok := ret.Get(0).(func([]byte) int); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
