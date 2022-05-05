package interfaces

import (
	"context"
)

// PostDeploymentTest defines a plugin that validates the health of a new deployment by
// executing a number of requests to it and then executing the
type PostDeploymentTest interface {
	Configurable

	// Execute the tests and return an error if the test fails
	Execute(ctx context.Context, candidateName string) error
}
