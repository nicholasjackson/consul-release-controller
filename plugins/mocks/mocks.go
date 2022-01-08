package mocks

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type Mocks struct {
	ReleaserMock *ReleaserMock
	RuntimeMock  *RuntimeMock
	MonitorMock  *MonitorMock
	StrategyMock *StrategyMock
}

// BuildMocks builds a mock provider and mock plugins for testing
func BuildMocks(t *testing.T) (*ProviderMock, *Mocks) {
	// create the mock plugins
	relMock := &ReleaserMock{}
	relMock.On("Configure", mock.Anything).Return(nil)
	relMock.On("Setup", mock.Anything).Return(nil)
	relMock.On("Scale", mock.Anything, mock.Anything).Return(nil)

	runMock := &RuntimeMock{}
	runMock.On("Configure", mock.Anything).Return(nil)
	runMock.On("Deploy", mock.Anything).Return(nil)
	runMock.On("Promote", mock.Anything).Return(nil)

	monMock := &MonitorMock{}
	monMock.On("Configure", mock.Anything).Return(nil)
	monMock.On("Check", mock.Anything).Return(nil)

	stratMock := &StrategyMock{}
	stratMock.On("Configure", mock.Anything).Return(nil)
	stratMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusSuccess, 10, nil)

	provMock := &ProviderMock{}
	provMock.On("CreateReleaser", mock.Anything).Return(relMock, nil)
	provMock.On("CreateRuntime", mock.Anything).Return(runMock, nil)
	provMock.On("CreateMonitor", mock.Anything).Return(monMock, nil)
	provMock.On("CreateStrategy", mock.Anything).Return(stratMock, nil)
	provMock.On("GetLogger", mock.Anything).Return(hclog.New(&hclog.LoggerOptions{Color: hclog.AutoColor, Level: hclog.Debug}))

	return provMock, &Mocks{relMock, runMock, monMock, stratMock}
}

// ProviderMock is a mock implementation of the provider that can be used for testing
type ProviderMock struct {
	mock.Mock
}

func (p *ProviderMock) CreateReleaser(pluginName string) (interfaces.Releaser, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Releaser), args.Error(1)
}

func (p *ProviderMock) CreateRuntime(pluginName string) (interfaces.Runtime, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Runtime), args.Error(1)
}

func (p *ProviderMock) CreateMonitor(pluginName string) (interfaces.Monitor, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Monitor), args.Error(1)
}

func (p *ProviderMock) CreateStrategy(pluginName string, mp interfaces.Monitor) (interfaces.Strategy, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Strategy), args.Error(1)
}

func (p *ProviderMock) GetLogger() hclog.Logger {
	args := p.Called()
	return args.Get(0).(hclog.Logger)
}
