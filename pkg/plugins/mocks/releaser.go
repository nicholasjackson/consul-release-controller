package mocks

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type ReleaserMock struct {
	mock.Mock
	baseConfig *interfaces.ReleaserBaseConfig
}

func (s *ReleaserMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := s.Called(data, log, store)

	s.baseConfig = &interfaces.ReleaserBaseConfig{}
	json.Unmarshal(data, s.baseConfig)

	return args.Error(0)
}

func (s *ReleaserMock) BaseConfig() interfaces.ReleaserBaseConfig {
	args := s.Called()
	if bc, ok := args.Get(0).(interfaces.ReleaserBaseConfig); ok {
		return bc
	}

	return *s.baseConfig
}

func (s *ReleaserMock) Setup(ctx context.Context, primarySubset, candidateSubset string) error {
	args := s.Called(ctx, primarySubset, candidateSubset)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (s *ReleaserMock) Scale(ctx context.Context, value int) error {
	args := s.Called(ctx, value)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (s *ReleaserMock) Destroy(ctx context.Context) error {
	args := s.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (s *ReleaserMock) WaitUntilServiceHealthy(ctx context.Context, filter string) error {
	args := s.Called(ctx, filter)

	return args.Error(0)
}
