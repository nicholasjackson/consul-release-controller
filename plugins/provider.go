package plugins

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/consul"
	"github.com/nicholasjackson/consul-canary-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-canary-controller/plugins/prometheus"
)

const (
	PluginReleaserTypeConsul    = "consul"
	PluginRuntimeTypeKubernetes = "kubernetes"
)

// Provider loads and creates registered plugins
type Provider interface {
	// CreateReleaser returns a Setup plugin that corresponds sto the given name
	CreateReleaser(pluginName string) (Releaser, error)

	// CreateRuntime returns a Runtime plugin that corresponds to the given name
	CreateRuntime(pluginName string) (Runtime, error)

	// CreateMonitoring returns a Monitor plugin that corresponds to the given name
	CreateMonitor(pluginName string) (Monitor, error)

	// CreateStrategy returns a Strategy plugin that corresponds to the given name
	CreateStrategy(pluginName string) (Strategy, error)
}

var prov Provider

// GetProvider lazy instantiates a plugin provider and returns a reference
func GetProvider() Provider {
	if prov == nil {
		prov = &ProviderImpl{hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Color: hclog.AutoColor})}
	}

	return prov
}

// ProviderImpl is the concrete implementation of the Provider interface
type ProviderImpl struct {
	log hclog.Logger
}

func (p *ProviderImpl) CreateReleaser(pluginName string) (Releaser, error) {
	p.log.Debug("Creating setup plugin", "name", pluginName)

	return consul.New(p.log.Named("consul-plugin"))
}

func (p *ProviderImpl) CreateRuntime(pluginName string) (Runtime, error) {
	if pluginName == "kubernetes" {
		return kubernetes.New(p.log.Named("kubernetes-plugin"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateMonitor(pluginName string) (Monitor, error) {
	if pluginName == "prometheus" {
		return prometheus.New(p.log.Named("prometheus-plugin"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateStrategy(pluginName string) (Strategy, error) {
	return nil, fmt.Errorf("not implemented")
}
