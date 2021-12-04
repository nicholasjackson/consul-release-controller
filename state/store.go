package state

import "github.com/nicholasjackson/consul-canary-controller/models"

type ListOptions struct {
	Runtime string // the type of the runtime i.e kubernetes
}

type Store interface {
	UpsertRelease(d *models.Release) error
	// ListReleases returns the releases in the data store that match the given options
	// if options is nil then all releases are returned
	ListReleases(options *ListOptions) ([]*models.Release, error)
	DeleteRelease(name string) error
}
