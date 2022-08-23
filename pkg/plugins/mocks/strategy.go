package mocks

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type StrategyMock struct {
	mock.Mock
}

func (r *StrategyMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := r.Called(data, log, store)

	return args.Error(0)
}

func (r *StrategyMock) Execute(ctx context.Context, candidateName string) (interfaces.StrategyStatus, int, error) {
	args := r.Called(ctx, candidateName)

	return interfaces.StrategyStatus(args.Get(0).(string)), args.Get(1).(int), args.Error(2)
}

func (p *StrategyMock) GetPrimaryTraffic() int {
	return p.Called().Int(0)
}

func (p *StrategyMock) GetCandidateTraffic() int {
	return p.Called().Int(0)
}
