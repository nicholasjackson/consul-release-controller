package plugins

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/canary"
	"github.com/nicholasjackson/consul-canary-controller/plugins/consul"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-canary-controller/plugins/prometheus"
)

var prov interfaces.Provider

// GetProvider lazy instantiates a plugin provider and returns a reference
func GetProvider(log hclog.Logger) interfaces.Provider {
	if prov == nil {
		prov = &ProviderImpl{log}
	}

	return prov
}

// ProviderImpl is the concrete implementation of the Provider interface
type ProviderImpl struct {
	log hclog.Logger
}

func (p *ProviderImpl) CreateReleaser(pluginName string) (interfaces.Releaser, error) {
	return consul.New(p.log.Named("consul-plugin"))
}

func (p *ProviderImpl) CreateRuntime(pluginName string) (interfaces.Runtime, error) {
	if pluginName == PluginRuntimeTypeKubernetes {
		return kubernetes.New(p.log.Named("kubernetes-plugin"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateMonitor(pluginName string) (interfaces.Monitor, error) {
	if pluginName == PluginMonitorTypePromethus {
		return prometheus.New(p.log.Named("prometheus-plugin"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateStrategy(pluginName string, mp interfaces.Monitor) (interfaces.Strategy, error) {
	if pluginName == PluginStrategyTypeCanary {
		return canary.New(p.log.Named("canary-strategy-plugin"), mp)
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) GetLogger() hclog.Logger {
	return p.log
}
