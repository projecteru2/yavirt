// Code generated by mockery v2.26.1. DO NOT EDIT.

package mocks

import (
	agent "github.com/projecteru2/yavirt/internal/virt/agent"
	domain "github.com/projecteru2/yavirt/internal/virt/domain"

	mock "github.com/stretchr/testify/mock"

	models "github.com/projecteru2/yavirt/internal/models"
)

// Bot is an autogenerated mock type for the Bot type
type Bot struct {
	mock.Mock
}

// Alloc provides a mock function with given fields:
func (_m *Bot) Alloc() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AllocFromImage provides a mock function with given fields: _a0
func (_m *Bot) AllocFromImage(_a0 models.Image) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(models.Image) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Amplify provides a mock function with given fields: delta, dom, ga, devPath
func (_m *Bot) Amplify(delta int64, dom domain.Domain, ga agent.Interface, devPath string) error {
	ret := _m.Called(delta, dom, ga, devPath)

	var r0 error
	if rf, ok := ret.Get(0).(func(int64, domain.Domain, agent.Interface, string) error); ok {
		r0 = rf(delta, dom, ga, devPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Check provides a mock function with given fields:
func (_m *Bot) Check() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *Bot) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CommitSnapshot provides a mock function with given fields: _a0
func (_m *Bot) CommitSnapshot(_a0 *models.Snapshot) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Snapshot) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConvertUserImage provides a mock function with given fields: user, name
func (_m *Bot) ConvertUserImage(user string, name string) (*models.UserImage, error) {
	ret := _m.Called(user, name)

	var r0 *models.UserImage
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*models.UserImage, error)); ok {
		return rf(user, name)
	}
	if rf, ok := ret.Get(0).(func(string, string) *models.UserImage); ok {
		r0 = rf(user, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*models.UserImage)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(user, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateSnapshot provides a mock function with given fields: _a0
func (_m *Bot) CreateSnapshot(_a0 *models.Snapshot) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Snapshot) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteAllSnapshots provides a mock function with given fields:
func (_m *Bot) DeleteAllSnapshots() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSnapshot provides a mock function with given fields: _a0
func (_m *Bot) DeleteSnapshot(_a0 *models.Snapshot) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Snapshot) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Mount provides a mock function with given fields: ga, devPath
func (_m *Bot) Mount(ga agent.Interface, devPath string) error {
	ret := _m.Called(ga, devPath)

	var r0 error
	if rf, ok := ret.Get(0).(func(agent.Interface, string) error); ok {
		r0 = rf(ga, devPath)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Repair provides a mock function with given fields:
func (_m *Bot) Repair() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RestoreSnapshot provides a mock function with given fields: _a0
func (_m *Bot) RestoreSnapshot(_a0 *models.Snapshot) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*models.Snapshot) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Undefine provides a mock function with given fields:
func (_m *Bot) Undefine() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewBot interface {
	mock.TestingT
	Cleanup(func())
}

// NewBot creates a new instance of Bot. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBot(t mockConstructorTestingTNewBot) *Bot {
	mock := &Bot{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
