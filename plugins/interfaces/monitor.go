package interfaces

import (
	"context"
	"time"
)

// Monitor defines an interface that all Monitoring platforms like Prometheus must implement
type Monitor interface {
	Configurable

	// Check the defined metrics to see that they are in tolerance
	Check(ctx context.Context, interval time.Duration) error
}
