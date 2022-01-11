package mocks

import (
	"context"
	"encoding/json"

	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
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

func (r *RuntimeMock) Deploy(ctx context.Context) error {
	args := r.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (r *RuntimeMock) Promote(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}

func (r *RuntimeMock) Destroy(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}
