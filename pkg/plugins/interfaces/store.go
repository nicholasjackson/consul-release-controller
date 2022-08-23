package interfaces

import (
	"errors"

	"github.com/nicholasjackson/consul-release-controller/pkg/models"
)

var ReleaseNotFound = errors.New("release not found")
var PluginStateNotFound = errors.New("plugin state not found")

// ListOptions are used for querying releases
type ListOptions struct {
	Runtime string // the type of the runtime i.e kubernetes
}

// Store defines a datastore plugin
type Store interface {
	ReleaseStore
	PluginStateStore
}

type ReleaseStore interface {
	// UpsertRelease creates a new release if not already existing, or updates and existing release
	UpsertRelease(r *models.Release) error

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

	// CreatePluginStateStore creates a plugin state store for the named plugin and release
	CreatePluginStateStore(r *models.Release, pluginName string) PluginStateStore
}

type PluginStateStore interface {
	// UpsertState creates a new state for the plugin if it does not already exist, or updates and existing plugin state
	UpsertState(data []byte) error

	// GetState with the given name
	// Returns a nil byte array and PluginStateNotFound error when state for the given plugin
	// does not exist in the store.
	// Any other error indicates an internal problem fetching the state
	GetState() ([]byte, error)
}
