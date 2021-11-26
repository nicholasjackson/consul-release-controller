package state

import (
	"errors"
	"sync"

	"github.com/nicholasjackson/consul-canary-controller/models"
)

var DeploymentNotFound = errors.New("Deployment not found")

type InmemStore struct {
	m           sync.Mutex
	deployments []*models.Deployment
}

func NewInmemStore() *InmemStore {
	return &InmemStore{m: sync.Mutex{}, deployments: []*models.Deployment{}}
}

func (m *InmemStore) UpsertDeployment(d *models.Deployment) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.deployments = append(m.deployments, d)

	return nil
}

func (m *InmemStore) ListDeployments() ([]*models.Deployment, error) {
	m.m.Lock()
	defer m.m.Unlock()

	return m.deployments, nil
}

func (m *InmemStore) DeleteDeployment(name string) error {
	m.m.Lock()
	defer m.m.Unlock()

	index := -1
	// find the correct deployment
	for i, d := range m.deployments {
		if d.ConsulService == name {
			index = i
			break
		}
	}

	m.deployments = append(m.deployments[:index], m.deployments[index+1:]...)

	if index == -1 {
		return DeploymentNotFound
	}

	return nil
}
