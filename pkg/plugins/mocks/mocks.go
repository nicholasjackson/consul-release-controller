package mocks

import (
	"bytes"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type Mocks struct {
	ReleaserMock       *ReleaserMock
	RuntimeMock        *RuntimeMock
	MonitorMock        *MonitorMock
	StrategyMock       *StrategyMock
	PostDeploymentMock *PostDeploymentTestMock
	MetricsMock        *MetricsMock
	StoreMock          *StoreMock
	StateMachineMock   *StateMachineMock
	WebhookMock        *WebhookMock
	LogBuffer          *bytes.Buffer
}

// BuildMocks builds a mock provider and mock plugins for testing
func BuildMocks(t *testing.T) (*ProviderMock, *Mocks) {
	// create the mock plugins
	relMock := &ReleaserMock{}
	relMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	relMock.On("BaseConfig").Return(nil)
	relMock.On("Setup", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	relMock.On("Scale", mock.Anything, mock.Anything).Return(nil)
	relMock.On("Destroy", mock.Anything, mock.Anything).Return(nil)
	relMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(nil)

	runMock := &RuntimeMock{}
	runMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	runMock.On("BaseConfig").Return(interfaces.RuntimeBaseConfig{DeploymentSelector: "api-(.*)", Namespace: "default"})
	runMock.On("BaseState").Return(interfaces.RuntimeBaseState{CandidateName: "api-deployment-v1", PrimaryName: "api-deployment"})
	runMock.On("InitPrimary", mock.Anything, mock.Anything).Return(interfaces.RuntimeDeploymentUpdate, nil)
	runMock.On("PromoteCandidate", mock.Anything).Return(interfaces.RuntimeDeploymentUpdate, nil)
	runMock.On("RemoveCandidate", mock.Anything).Return(nil)
	runMock.On("RestoreOriginal", mock.Anything).Return(nil)
	runMock.On("RemovePrimary", mock.Anything).Return(nil)
	runMock.On("CandidateSubsetFilter").Return(nil)
	runMock.On("PrimarySubsetFilter").Return(nil)

	monMock := &MonitorMock{}
	monMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	monMock.On("Check", mock.Anything, mock.Anything, mock.Anything).Return(interfaces.CheckSuccess, nil)

	stratMock := &StrategyMock{}
	stratMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	stratMock.On("Execute", mock.Anything, mock.Anything).Return(interfaces.StrategyStatusSuccess, 10, nil)
	stratMock.On("GetPrimaryTraffic", mock.Anything).Return(40)
	stratMock.On("GetCandidateTraffic", mock.Anything).Return(60)

	metricsMock := &MetricsMock{}
	metricsMock.On("ServiceStarting")
	metricsMock.On("HandleRequest", mock.Anything, mock.Anything).Return(func(status int) {})
	metricsMock.On("StateChanged", mock.Anything, mock.Anything, mock.Anything).Return(func(status int) {})

	stateMock := &StateMachineMock{}
	stateMock.On("Configure").Return(nil)
	stateMock.On("Deploy").Return(nil)
	stateMock.On("Destroy").Return(nil)
	stateMock.On("CurrentState").Return(interfaces.StateStart)

	storeMock := &StoreMock{}
	storeMock.On("UpsertRelease", mock.Anything).Return(nil)
	storeMock.On("ListReleases", mock.Anything).Return(nil, nil)
	storeMock.On("DeleteRelease", mock.Anything).Return(nil)
	storeMock.On("GetRelease", mock.Anything).Return(nil, nil)
	storeMock.On("CreatePluginStateStore", mock.Anything, mock.Anything).Return(storeMock)
	storeMock.On("UpsertState", mock.Anything).Return(nil)
	storeMock.On("GetState").Return(nil, nil)

	webhookMock := &WebhookMock{}
	webhookMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	webhookMock.On("Send", mock.Anything, mock.Anything).Return(nil)

	postDeploymentMock := &PostDeploymentTestMock{}
	postDeploymentMock.On("Configure", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	postDeploymentMock.On("Execute", mock.Anything, mock.Anything).Return(nil)

	provMock := &ProviderMock{}

	provMock.On("CreateReleaser", mock.Anything).Return(relMock, nil)
	provMock.On("CreateRuntime", mock.Anything).Return(runMock, nil)
	provMock.On("CreateMonitor", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(monMock, nil)
	provMock.On("CreateStrategy", mock.Anything).Return(stratMock, nil)
	provMock.On("CreateWebhook", mock.Anything).Return(webhookMock, nil)
	provMock.On("CreatePostDeploymentTest", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(postDeploymentMock, nil)

	logBuffer := bytes.NewBufferString("")

	logger := hclog.New(
		&hclog.LoggerOptions{
			Level:  hclog.Trace,
			Output: logBuffer,
		},
	)

	provMock.On("GetLogger", mock.Anything).Return(logger)
	provMock.On("GetMetrics").Return(metricsMock)
	provMock.On("GetDataStore").Return(storeMock)
	provMock.On("GetStateMachine", mock.Anything).Return(stateMock, nil)
	provMock.On("DeleteStateMachine", mock.Anything).Return(nil)

	return provMock, &Mocks{relMock, runMock, monMock, stratMock, postDeploymentMock, metricsMock, storeMock, stateMock, webhookMock, logBuffer}
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

func (p *ProviderMock) CreateMonitor(pluginName, deploymentName, namespace, runtime string) (interfaces.Monitor, error) {
	args := p.Called(pluginName, deploymentName, namespace, runtime)

	return args.Get(0).(interfaces.Monitor), args.Error(1)
}

func (p *ProviderMock) CreateStrategy(pluginName string, mp interfaces.Monitor) (interfaces.Strategy, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Strategy), args.Error(1)
}

func (p *ProviderMock) CreateWebhook(pluginName string) (interfaces.Webhook, error) {
	args := p.Called(pluginName)

	return args.Get(0).(interfaces.Webhook), args.Error(1)
}

func (p *ProviderMock) CreatePostDeploymentTest(pluginName, deploymentName, namespace, runtime string, mp interfaces.Monitor) (interfaces.PostDeploymentTest, error) {
	args := p.Called(pluginName, deploymentName, namespace, runtime, mp)

	return args.Get(0).(interfaces.PostDeploymentTest), args.Error(1)
}

func (p *ProviderMock) GetRuntimeClient(runtime string) (interfaces.RuntimeClient, error) {
	args := p.Called(runtime)

	if rc, ok := args.Get(0).(interfaces.RuntimeClient); ok {
		return rc, args.Error(1)
	}

	return nil, args.Error(1)
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
