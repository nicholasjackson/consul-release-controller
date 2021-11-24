package state

import "github.com/nicholasjackson/consul-canary-controller/models"

type Store interface {
	SetDeployment(d *models.Deployment) error
}
