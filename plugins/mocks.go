package plugins

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/mock"
)

type Mocks struct {
	ReleaserMock *releaserMock
	RuntimeMock  *runtimeMock
	MonitorMock  *monitorMock
}

// BuildMocks builds a mock provider and mock plugins for testing
func BuildMocks(t *testing.T) (*ProviderMock, *Mocks) {
	// create the mock plugins
	sm := &releaserMock{}
	sm.On("Configure", mock.Anything).Return(nil)
	sm.On("Setup", mock.Anything, mock.Anything).Return(nil)

	rm := &runtimeMock{}
	rm.On("Configure", mock.Anything).Return(nil)
	rm.On("Deploy", mock.Anything, mock.Anything).Return(nil)

	mm := &monitorMock{}
	rm.On("Configure", mock.Anything).Return(nil)
	rm.On("Check", mock.Anything).Return(nil)

	mp := &ProviderMock{}
	mp.On("CreateReleaser", mock.Anything).Return(sm, nil)
	mp.On("CreateRuntime", mock.Anything).Return(rm, nil)
	mp.On("CreateMonitor", mock.Anything).Return(mm, nil)

	return mp, &Mocks{sm, rm, mm}
}

// ProviderMock is a mock implementation of the provider that can be used for testing
type ProviderMock struct {
	mock.Mock
}

func (p *ProviderMock) CreateReleaser(pluginName string) (Releaser, error) {
	args := p.Called(pluginName)

	return args.Get(0).(Releaser), args.Error(1)
}

func (p *ProviderMock) CreateRuntime(pluginName string) (Runtime, error) {
	args := p.Called(pluginName)

	return args.Get(0).(Runtime), args.Error(1)
}

func (p *ProviderMock) CreateMonitor(pluginName string) (Monitor, error) {
	args := p.Called(pluginName)

	return args.Get(0).(Monitor), args.Error(1)
}

func (p *ProviderMock) CreateStrategy(pluginName string) (Strategy, error) {
	args := p.Called(pluginName)

	return args.Get(0).(Strategy), args.Error(1)
}

type releaserMock struct {
	mock.Mock
}

func (s *releaserMock) Setup(ctx context.Context) error {
	args := s.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (s *releaserMock) Configure(config json.RawMessage) error {
	args := s.Called(config)

	return args.Error(0)
}

func (s *releaserMock) Scale(ctx context.Context, value int) error {
	args := s.Called(ctx, value)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (s *releaserMock) Destroy(ctx context.Context) error {
	args := s.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

type runtimeMock struct {
	mock.Mock
}

func (r *runtimeMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *runtimeMock) GetConfig() interface{} {
	args := r.Called()

	return args.Get(0)
}

func (r *runtimeMock) Deploy(ctx context.Context) error {
	args := r.Called(ctx)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return nil
}

func (r *runtimeMock) Destroy(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}

type monitorMock struct {
	mock.Mock
}

func (r *monitorMock) Configure(c json.RawMessage) error {
	args := r.Called(c)

	return args.Error(0)
}

func (r *monitorMock) Check(ctx context.Context) error {
	args := r.Called(ctx)

	return args.Error(0)
}
