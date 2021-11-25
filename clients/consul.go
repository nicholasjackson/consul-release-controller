package clients

import "github.com/stretchr/testify/mock"

type Consul interface {
	CreateServiceDefaults(name string) error
	CreateServiceResolver(name string) error
	CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error
	CreateServiceRouter(name string) error
}

type MockConsul struct {
	mock.Mock
}

func (mc *MockConsul) CreateServiceDefaults(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *MockConsul) CreateServiceResolver(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *MockConsul) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	args := mc.Called(name, primaryTraffic, canaryTraffic)

	return args.Error(0)
}

func (mc *MockConsul) CreateServiceRouter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}
