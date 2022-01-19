package mocks

import (
	"context"
	"encoding/json"

	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type StrategyMock struct {
	mock.Mock
}

func (r *StrategyMock) Configure(name, namespace string, c json.RawMessage) error {
	args := r.Called(name, namespace, c)

	return args.Error(0)
}

func (r *StrategyMock) Execute(ctx context.Context) (interfaces.StrategyStatus, int, error) {
	args := r.Called(ctx)

	return interfaces.StrategyStatus(args.Get(0).(string)), args.Get(1).(int), args.Error(2)
}
