package state

import (
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) UpsertRelease(d *models.Release) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *MockStore) ListReleases(options *ListOptions) ([]*models.Release, error) {
	args := m.Called(options)

	var deps []*models.Release
	if d, ok := args.Get(0).([]*models.Release); ok {
		deps = d
	}

	return deps, args.Error(1)
}

func (m *MockStore) DeleteRelease(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockStore) GetRelease(name string) (*models.Release, error) {
	args := m.Called(name)

	if r, ok := args.Get(0).(*models.Release); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}
