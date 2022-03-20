package mocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stretchr/testify/mock"
)

// PostDeploymentTest defines a plugin that validates the health of a new deployment by
// executing a number of requests to it and then executing the
type PostDeploymentTestMock struct {
	mock.Mock
}

func (r *PostDeploymentTestMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *PostDeploymentTestMock) Execute(ctx context.Context, interval time.Duration) error {
	args := r.Called(ctx, interval)

	return args.Error(0)
}
