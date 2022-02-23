package interfaces

import (
	"context"
	"encoding/json"
)

type ServiceVariant int

const (
	All       ServiceVariant = 0
	Primary   ServiceVariant = 1
	Candidate ServiceVariant = 2
)

// Releaser defines methods for configuring and manipulating traffic in the service mesh
type Releaser interface {
	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// Setup the necessary configuration for the service mesh
	// Returning an error from this function will fail the deployment
	Setup(ctx context.Context) error

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
	WaitUntilServiceHealthy(ctx context.Context, t ServiceVariant) error
}
