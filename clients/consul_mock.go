package clients

import (
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type ConsulMock struct {
	mock.Mock
}

func (mc *ConsulMock) CreateServiceDefaults(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceResolver(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	args := mc.Called(name, primaryTraffic, canaryTraffic)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceRouter(name string, onlyDefault bool) error {
	args := mc.Called(name, onlyDefault)

	return args.Error(0)
}

func (mc *ConsulMock) DeleteServiceDefaults(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) DeleteServiceResolver(name string) error {
	args := mc.Called(name)

	return args.Error(0)

}

func (mc *ConsulMock) DeleteServiceSplitter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) DeleteServiceRouter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CheckHealth(name string, t interfaces.ServiceVariant) error {
	args := mc.Called(name)

	return args.Error(0)
}
