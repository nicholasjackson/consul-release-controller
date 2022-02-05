package mocks

import (
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
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
