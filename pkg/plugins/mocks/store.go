package mocks

import (
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type StoreMock struct {
	mock.Mock
}

func (m *StoreMock) UpsertRelease(d *models.Release) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *StoreMock) ListReleases(options *interfaces.ListOptions) ([]*models.Release, error) {
	args := m.Called(options)

	if d, ok := args.Get(0).([]*models.Release); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *StoreMock) DeleteRelease(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *StoreMock) GetRelease(name string) (*models.Release, error) {
	args := m.Called(name)

	if r, ok := args.Get(0).(*models.Release); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *StoreMock) CreatePluginStateStore(r *models.Release, pluginName string) interfaces.PluginStateStore {
	m.Called(r, pluginName)
	return m
}

func (m *StoreMock) UpsertState(data []byte) error {

	args := m.Called(data)
	return args.Error(0)

}

func (m *StoreMock) GetState() ([]byte, error) {

	args := m.Called()

	if d, ok := args.Get(0).([]byte); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}
