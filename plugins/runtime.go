package plugins

import (
	"context"
	"encoding/json"
)

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// Deploy the new test version to the platform
	Deploy(ctx context.Context, callback func()) error

	// Destroy removes any configuration that was created with the Deploy method
	Destroy(ctx context.Context, callback func()) error
}
