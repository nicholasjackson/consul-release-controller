package interfaces

import (
	"errors"

	"github.com/nicholasjackson/consul-release-controller/models"
)

var ReleaseNotFound = errors.New("Release not found")

// ListOptions are used for querying releases
type ListOptions struct {
	Runtime string // the type of the runtime i.e kubernetes
}

// Store defines a datastore plugin
type Store interface {
	// UpsertRelease creates a new release if not already existing, or updates and existing release
	UpsertRelease(d *models.Release) error

	// ListReleases returns the releases in the data store that match the given options
	// if options is nil then all releases are returned
	ListReleases(options *ListOptions) ([]*models.Release, error)

	// GetRelease with the given name
	// Returns a nil Release and ReleaseNotFound error when a Release with the given name does not
	// exist in the store.
	// Any other error indicates an internal problem fetching the Release
	GetRelease(name string) (*models.Release, error)

	// DeleteRelease with the given name
	DeleteRelease(name string) error
}
