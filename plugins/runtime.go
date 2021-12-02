package plugins

import (
	"context"
	"encoding/json"

	"github.com/stretchr/testify/mock"
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

type RuntimeMock struct {
	mock.Mock
}

func (r *RuntimeMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *RuntimeMock) Deploy(ctx context.Context, done func()) error {
	args := r.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	// call the callback if no error
	done()

	return nil
}

func (r *RuntimeMock) Destroy(ctx context.Context, callback func()) error {
	args := r.Called(ctx)

	return args.Error(0)
}
