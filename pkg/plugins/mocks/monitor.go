package mocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type MonitorMock struct {
	mock.Mock
}

func (r *MonitorMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := r.Called(data, log, store)

	return args.Error(0)
}

func (r *MonitorMock) Check(ctx context.Context, candidateName string, interval time.Duration) (interfaces.CheckResult, error) {
	args := r.Called(ctx, candidateName, interval)

	return args.Get(0).(interfaces.CheckResult), args.Error(1)
}
