package plugins

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/consul"
	"github.com/stretchr/testify/mock"
)

// Provider loads and creates registered plugins
type Provider interface {
	// CreateReleaser returns a Setup plugin that corresponds sto the given name
	CreateReleaser(pluginName string) (Releaser, error)

	// CreateRuntime returns a Runtime plugin that corresponds to the given name
	CreateRuntime(pluginName string) (Runtime, error)

	// CreateMonitoring returns a Runtime plugin that corresponds to the given name
	CreateMonitoring(name string) (Monitoring, error)
}

var prov Provider

// GetProvider lazy instantiates a plugin provider and returns a reference
func GetProvider() Provider {
	if prov == nil {
		prov = &ProviderImpl{hclog.Default()}
	}

	return prov
}

// ProviderImpl is the concrete implementation of the Provider interface
type ProviderImpl struct {
	log hclog.Logger
}

func (p *ProviderImpl) CreateReleaser(pluginName string) (Releaser, error) {
	p.log.Debug("Creating setup plugin", "name", pluginName)

	return consul.New(p.log), nil
}

func (p *ProviderImpl) CreateRuntime(pluginName string) (Runtime, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateMonitoring(pluginName string) (Monitoring, error) {
	return nil, fmt.Errorf("not implemented")
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

func (p *ProviderMock) CreateMonitoring(pluginName string) (Monitoring, error) {
	args := p.Called(pluginName)

	return args.Get(0).(Monitoring), args.Error(1)
}
