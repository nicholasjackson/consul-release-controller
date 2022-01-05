package plugins

import (
	"context"
	"encoding/json"
)

// Monitor defines an interface that all Monitoring platforms like Prometheus must implement
type Monitor interface {

	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// Check the defined metrics to see that they are in tolerance
	Check(ctx context.Context) error
}
