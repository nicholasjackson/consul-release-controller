package mocks

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type RuntimeMock struct {
	mock.Mock
}

func (r *RuntimeMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := r.Called(data, log, store)

	return args.Error(0)
}

func (r *RuntimeMock) BaseConfig() interfaces.RuntimeBaseConfig {
	args := r.Called()

	return args.Get(0).(interfaces.RuntimeBaseConfig)
}

func (r *RuntimeMock) BaseState() interfaces.RuntimeBaseState {
	args := r.Called()

	return args.Get(0).(interfaces.RuntimeBaseState)
}

func (r *RuntimeMock) InitPrimary(ctx context.Context, releaseName string) (interfaces.RuntimeDeploymentStatus, error) {
	args := r.Called(ctx, releaseName)

	return args.Get(0).(interfaces.RuntimeDeploymentStatus), args.Error(1)
}

func (r *RuntimeMock) PromoteCandidate(ctx context.Context) (interfaces.RuntimeDeploymentStatus, error) {
	args := r.Called(ctx)

	return args.Get(0).(interfaces.RuntimeDeploymentStatus), args.Error(1)
}

func (r *RuntimeMock) RemoveCandidate(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}

func (r *RuntimeMock) RestoreOriginal(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}

func (r *RuntimeMock) RemovePrimary(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
func (r *RuntimeMock) CandidateSubsetFilter() string {
	r.Called()
	return "candidate filter"
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
func (r *RuntimeMock) PrimarySubsetFilter() string {
	r.Called()
	return "primary filter"
}
