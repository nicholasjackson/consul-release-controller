package state

import (
	"sync"

	"github.com/nicholasjackson/consul-canary-controller/models"
)

type InmemStore struct {
	m           sync.Mutex
	deployments []models.Deployment
}

func NewInmemStore() *InmemStore {
	return &InmemStore{m: sync.Mutex{}, deployments: []models.Deployment{}}
}

func (m *InmemStore) UpsertDeployment(d models.Deployment) error {
	m.deployments = append(m.deployments, d)

	return nil
}
