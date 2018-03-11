package _mocks

import loader "kube-helper/loader"
import mock "github.com/stretchr/testify/mock"

// ApplicationServiceInterface is an autogenerated mock type for the ApplicationServiceInterface type
type ApplicationServiceInterface struct {
	mock.Mock
}

// Apply provides a mock function with given fields:
func (_m *ApplicationServiceInterface) Apply() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteByNamespace provides a mock function with given fields:
func (_m *ApplicationServiceInterface) DeleteByNamespace() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetDomain provides a mock function with given fields: dnsConfig
func (_m *ApplicationServiceInterface) GetDomain(dnsConfig loader.DNSConfig) string {
	ret := _m.Called(dnsConfig)

	var r0 string
	if rf, ok := ret.Get(0).(func(loader.DNSConfig) string); ok {
		r0 = rf(dnsConfig)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// HandleIngressAnnotationOnApply provides a mock function with given fields:
func (_m *ApplicationServiceInterface) HandleIngressAnnotationOnApply() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HasNamespace provides a mock function with given fields:
func (_m *ApplicationServiceInterface) HasNamespace() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
