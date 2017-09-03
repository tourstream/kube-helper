package service

import loader "kube-helper/loader"
import mock "github.com/stretchr/testify/mock"
import model "kube-helper/model"

// MockImagesInterface is an autogenerated mock type for the ImagesInterface type
type MockImagesInterface struct {
	mock.Mock
}

// DeleteManifest provides a mock function with given fields: config, manifest
func (_m *MockImagesInterface) DeleteManifest(config loader.Cleanup, manifest string) error {
	ret := _m.Called(config, manifest)

	var r0 error
	if rf, ok := ret.Get(0).(func(loader.Cleanup, string) error); ok {
		r0 = rf(config, manifest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HasTag provides a mock function with given fields: config, tag
func (_m *MockImagesInterface) HasTag(config loader.Cleanup, tag string) (bool, error) {
	ret := _m.Called(config, tag)

	var r0 bool
	if rf, ok := ret.Get(0).(func(loader.Cleanup, string) bool); ok {
		r0 = rf(config, tag)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(loader.Cleanup, string) error); ok {
		r1 = rf(config, tag)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// List provides a mock function with given fields: config
func (_m *MockImagesInterface) List(config loader.Cleanup) (*model.TagCollection, error) {
	ret := _m.Called(config)

	var r0 *model.TagCollection
	if rf, ok := ret.Get(0).(func(loader.Cleanup) *model.TagCollection); ok {
		r0 = rf(config)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.TagCollection)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(loader.Cleanup) error); ok {
		r1 = rf(config)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Untag provides a mock function with given fields: config, tag
func (_m *MockImagesInterface) Untag(config loader.Cleanup, tag string) error {
	ret := _m.Called(config, tag)

	var r0 error
	if rf, ok := ret.Get(0).(func(loader.Cleanup, string) error); ok {
		r0 = rf(config, tag)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}