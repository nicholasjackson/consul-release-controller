package interfaces

import (
	"context"
	"encoding/json"
	"time"
)

// Monitor defines an interface that all Monitoring platforms like Prometheus must implement
type Monitor interface {

	// Configure the plugin with the given json
	Configure(name, namespace, runtime string, config json.RawMessage) error

	// Check the defined metrics to see that they are in tolerance
	Check(ctx context.Context, interval time.Duration) error
}
