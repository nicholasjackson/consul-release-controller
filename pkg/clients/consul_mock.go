package clients

import (
	"github.com/stretchr/testify/mock"
)

type ConsulMock struct {
	mock.Mock
}

func (mc *ConsulMock) CreateServiceDefaults(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceResolver(name, primarySubsetFilter, candidateSubsetFilter string) error {
	args := mc.Called(name, primarySubsetFilter, candidateSubsetFilter)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceSplitter(name string, primaryTraffic, canaryTraffic int) error {
	args := mc.Called(name, primaryTraffic, canaryTraffic)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceRouter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CreateUpstreamRouter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CreateServiceIntention(name string) error {
	args := mc.Called(name)

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

func (mc *ConsulMock) DeleteUpstreamRouter(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) DeleteServiceIntention(name string) error {
	args := mc.Called(name)

	return args.Error(0)
}

func (mc *ConsulMock) CheckHealth(name, filter string) error {
	args := mc.Called(name, filter)

	return args.Error(0)
}

// SetKV sets the data at the given path in the Consul Key Value store
func (mc *ConsulMock) SetKV(path string, data []byte) error {
	args := mc.Mock.Called(path, data)

	return args.Error(0)
}

// GetKV gets the data at the given path in the Consul Key Value store
func (mc *ConsulMock) GetKV(path string) ([]byte, error) {
	args := mc.Mock.Called(path)

	if d, ok := args.Get(0).([]byte); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

// DeleteKV deletes the data at the given path in the Consul Key Value store
func (mc *ConsulMock) DeleteKV(path string) error {
	args := mc.Mock.Called(path)

	return args.Error(0)
}

func (mc *ConsulMock) ListKV(path string) ([]string, error) {
	args := mc.Mock.Called(path)
	if d, ok := args.Get(0).([]string); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}
