package mocks

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

// PostDeploymentTest defines a plugin that validates the health of a new deployment by
// executing a number of requests to it and then executing the
type PostDeploymentTestMock struct {
	mock.Mock
}

func (r *PostDeploymentTestMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := r.Called(data, log, store)

	return args.Error(0)
}

func (r *PostDeploymentTestMock) Execute(ctx context.Context, candidateName string) error {
	args := r.Called(ctx, candidateName)

	return args.Error(0)
}
