package state

import "github.com/nicholasjackson/consul-canary-controller/models"

type Store interface {
	UpsertDeployment(d *models.Deployment) error
}
