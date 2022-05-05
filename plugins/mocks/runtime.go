package mocks

import (
	"context"
	"encoding/json"

	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type RuntimeMock struct {
	mock.Mock
}

func (r *RuntimeMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *RuntimeMock) BaseConfig() interfaces.RuntimeBaseConfig {
	args := r.Called()

	return args.Get(0).(interfaces.RuntimeBaseConfig)
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
