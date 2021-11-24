package state

import (
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) SetDeployment(d *models.Deployment) error {
	args := m.Called(d)
	return args.Error(0)
}
