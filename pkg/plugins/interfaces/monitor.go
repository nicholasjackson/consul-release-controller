package interfaces

import (
	"context"
	"time"
)

type CheckResult int

const (
	CheckSuccess CheckResult = iota
	CheckFailed
	CheckNoMetrics
	CheckError
)

// Monitor defines an interface that all Monitoring platforms like Prometheus must implement
type Monitor interface {
	Configurable

	// Check the defined metrics to see that they are in tolerance, if the check fails
	// either due to an internal error, failed precondition, or no data, a detailed
	// error is returned. In all instances CheckResult is returned informing
	// the caller of the outcome.
	//
	// CheckSuccess   - The check has completed successfully.
	// CheckFailed    - The call was successfully made to the metrics database but the result
	//                  was not in tolerance.
	// CheckNoMetrics - The check completed successfully but no data was returned from the metrics db.
	// CheckError     - An internal error occurred.
	Check(ctx context.Context, candidateName string, interval time.Duration) (CheckResult, error)
}
