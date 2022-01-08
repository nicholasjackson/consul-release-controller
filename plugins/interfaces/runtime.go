package interfaces

import (
	"context"
	"encoding/json"
)

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// GetConfig returns the plugin config
	// this is returned as an interface as every Runtime plugin can
	// have different config
	GetConfig() interface{}

	// Deploy the new test version to the platform
	Deploy(ctx context.Context) error

	// Promote the new test version to primary
	Promote(ctx context.Context) error

	// Destroy removes any configuration that was created with the Deploy method
	Destroy(ctx context.Context) error
}
