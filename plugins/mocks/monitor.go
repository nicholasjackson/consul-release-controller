package mocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type MonitorMock struct {
	mock.Mock
}

func (r *MonitorMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *MonitorMock) Check(ctx context.Context, candidateName string, interval time.Duration) (interfaces.CheckResult, error) {
	args := r.Called(ctx, candidateName, interval)

	return args.Get(0).(interfaces.CheckResult), args.Error(1)
}
