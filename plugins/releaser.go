package plugins

import (
	"context"
	"encoding/json"
)

// Releaser defines methods for configuring and manipulating traffic in the service mesh
type Releaser interface {
	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// Setup the necessary configuration for the service mesh
	// On successful completion the statemachine expects that the done function
	// is called in order to progress to the next step.
	// Returning an error from this function will fail the deployment
	Setup(ctx context.Context, done func()) error

	// Scale sets the percentage of traffic that is distributed to the canary instance
	//
	// e.g. if value is 90, then the percentage of traffic to the Canary will be 90% and
	// the Primary would be 10%
	Scale(ctx context.Context, value int, done func()) error

	// Destroy removes any configuration that was created with the Configure method
	Destroy(ctx context.Context, done func()) error
}
