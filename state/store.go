package state

import "github.com/nicholasjackson/consul-canary-controller/models"

type ListOptions struct {
	Runtime string // the type of the runtime i.e kubernetes
}

type Store interface {
	// UpsertRelease creates a new release if not already existing, or updates and existing release
	UpsertRelease(d *models.Release) error

	// ListReleases returns the releases in the data store that match the given options
	// if options is nil then all releases are returned
	ListReleases(options *ListOptions) ([]*models.Release, error)

	// GetRelease with the given name
	GetRelease(name string) (*models.Release, error)

	// DeleteRelease with the given name
	DeleteRelease(name string) error
}
