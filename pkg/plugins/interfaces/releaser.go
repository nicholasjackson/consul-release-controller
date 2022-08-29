package interfaces

import (
	"context"
)

type ServiceVariant int

const (
	All       ServiceVariant = 0
	Primary   ServiceVariant = 1
	Candidate ServiceVariant = 2
)

type ReleaserBaseConfig struct {
	ConsulService string `json:"consul_service" validate:"required"`
	Namespace     string `json:"namespace"`
	Partition     string `json:"partition"`
}

// Releaser defines methods for configuring and manipulating traffic in the service mesh
type Releaser interface {
	Configurable

	// BaseConfig returns the base Runtime config
	// all Runtime plugins should embed RuntimeBaseConfig in their own config
	BaseConfig() ReleaserBaseConfig

	// Setup the necessary configuration for the service mesh
	// Returning an error from this function will fail the deployment
	Setup(ctx context.Context, primarySubsetFilter, candidateSubsetFilter string) error

	// Scale sets the percentage of traffic that is distributed to the canary instance
	//
	// e.g. if value is 90, then the percentage of traffic to the Canary will be 90% and
	// the Primary would be 10%
	Scale(ctx context.Context, value int) error

	// Destroy removes any configuration that was created with the Configure method
	Destroy(ctx context.Context) error

	// WaitUntilServiceHealthy blocks until the service mesh service is passing all it's
	// health checks. Returns an error if any endpoints for a service have failing health
	// checks.
	// filter allows the filtering of service instances
	WaitUntilServiceHealthy(ctx context.Context, filter string) error
}
