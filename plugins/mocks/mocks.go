package mocks

import (
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type Mocks struct {
	ReleaserMock     *ReleaserMock
	RuntimeMock      *RuntimeMock
	MonitorMock      *MonitorMock
	StrategyMock     *StrategyMock
	MetricsMock      *MetricsMock
	StoreMock        *StoreMock
	StateMachineMock *StateMachineMock
}

// BuildMocks builds a mock provider and mock plugins for testing
func BuildMocks(t *testing.T) (*ProviderMock, *Mocks) {
	// create the mock plugins
	relMock := &ReleaserMock{}
	relMock.On("Configure", mock.Anything).Return(nil)
	relMock.On("Setup", mock.Anything).Return(nil)
	relMock.On("Scale", mock.Anything, mock.Anything).Return(nil)
	relMock.On("Destroy", mock.Anything, mock.Anything).Return(nil)
	relMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(nil)

	runMock := &RuntimeMock{}
	runMock.On("Configure", mock.Anything).Return(nil)
	runMock.On("BaseConfig").Return(interfaces.RuntimeBaseConfig{Deployment: "api-deployment", Namespace: "default"})
	runMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentUpdate, nil)
	runMock.On("PromoteCandidate", mock.Anything).Return(interfaces.RuntimeDeploymentUpdate, nil)
	runMock.On("RemoveCandidate", mock.Anything).Return(nil)
	runMock.On("RestoreOriginal", mock.Anything).Return(nil)
	runMock.On("RemovePrimary", mock.Anything).Return(nil)

	monMock := &MonitorMock{}
	monMock.On("Configure", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	monMock.On("Check", mock.Anything, mock.Anything).Return(nil)

	stratMock := &StrategyMock{}
	stratMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	stratMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusSuccess, 10, nil)

	metricsMock := &MetricsMock{}
	metricsMock.On("ServiceStarting")
	metricsMock.On("HandleRequest", mock.Anything, mock.Anything).Return(func(status int) {})
	metricsMock.On("StateChanged", mock.Anything, mock.Anything, mock.Anything).Return(func(status int) {})

	stateMock := &StateMachineMock{}
	stateMock.On("Configure").Return(nil)
	stateMock.On("Deploy").Return(nil)
	stateMock.On("Destroy").Return(nil)
	stateMock.On("CurrentState").Return(interfaces.StateStart)
	stateMock.On("StateHistory").Return([]interfaces.StateHistory{interfaces.StateHistory{Time: time.Now(), State: interfaces.StateStart}})

	storeMock := &StoreMock{}
	storeMock.On("UpsertRelease", mock.Anything).Return(nil)
	storeMock.On("ListReleases", mock.Anything).Return(nil, nil)
	storeMock.On("DeleteRelease", mock.Anything).Return(nil)
	storeMock.On("GetRelease", mock.Anything).Return(nil, nil)

	provMock := &ProviderMock{}
	provMock.On("CreateReleaser", mock.Anything).Return(relMock, nil)
	provMock.On("CreateRuntime", mock.Anything).Return(runMock, nil)
	provMock.On("CreateMonitor", mock.Anything).Return(monMock, nil)
	provMock.On("CreateStrategy", mock.Anything).Return(stratMock, nil)
	provMock.On("GetLogger", mock.Anything).Return(hclog.New(&hclog.LoggerOptions{Color: hclog.AutoColor, Level: hclog.Debug}))
	provMock.On("GetMetrics").Return(metricsMock)
	provMock.On("GetDataStore").Return(storeMock)
	provMock.On("GetStateMachine", mock.Anything).Return(stateMock, nil)
	provMock.On("DeleteStateMachine", mock.Anything).Return(nil)

	return provMock, &Mocks{relMock, runMock, monMock, stratMock, metricsMock, storeMock, stateMock}
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

func (p *ProviderMock) GetMetrics() interfaces.Metrics {
	args := p.Called()
	return args.Get(0).(interfaces.Metrics)
}

func (p *ProviderMock) GetDataStore() interfaces.Store {
	args := p.Called()
	return args.Get(0).(interfaces.Store)
}

func (p *ProviderMock) GetStateMachine(release *models.Release) (interfaces.StateMachine, error) {
	args := p.Called(release)

	if r, ok := args.Get(0).(interfaces.StateMachine); ok {
		return r, args.Error(1)
	}

	return nil, args.Error(1)
}

func (p *ProviderMock) DeleteStateMachine(release *models.Release) error {
	args := p.Called(release)

	return args.Error(0)
}
