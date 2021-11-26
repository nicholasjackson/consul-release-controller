package state

import (
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) UpsertDeployment(d *models.Deployment) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *MockStore) ListDeployments() ([]*models.Deployment, error) {
	args := m.Called()

	var deps []*models.Deployment
	if d, ok := args.Get(0).([]*models.Deployment); ok {
		deps = d
	}

	return deps, args.Error(1)
}

func (m *MockStore) DeleteDeployment(name string) error {
	args := m.Called(name)
	return args.Error(0)
}
