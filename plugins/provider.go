package plugins

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/canary"
	"github.com/nicholasjackson/consul-release-controller/plugins/consul"
	"github.com/nicholasjackson/consul-release-controller/plugins/discord"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-release-controller/plugins/prometheus"
	"github.com/nicholasjackson/consul-release-controller/plugins/statemachine"
)

var prov interfaces.Provider

// temporarily store the active statemachines here, at some point we need to
// think about serializing state
var statemachines map[*models.Release]interfaces.StateMachine

// GetProvider lazy instantiates a plugin provider and returns a reference
func GetProvider(log hclog.Logger, metrics interfaces.Metrics, store interfaces.Store) interfaces.Provider {
	if prov == nil {
		statemachines = map[*models.Release]interfaces.StateMachine{}
		prov = &ProviderImpl{log, metrics, store}
	}

	return prov
}

// ProviderImpl is the concrete implementation of the Provider interface
type ProviderImpl struct {
	log     hclog.Logger
	metrics interfaces.Metrics
	store   interfaces.Store
}

func (p *ProviderImpl) CreateReleaser(pluginName string) (interfaces.Releaser, error) {
	return consul.New(p.log.Named("releaser-plugin-consul"))
}

func (p *ProviderImpl) CreateRuntime(pluginName string) (interfaces.Runtime, error) {
	if pluginName == PluginRuntimeTypeKubernetes {
		return kubernetes.New(p.log.Named("runtime-plugin-kubernetes"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateMonitor(pluginName, name, namespace, runtime string) (interfaces.Monitor, error) {
	if pluginName == PluginMonitorTypePromethus {
		return prometheus.New(name, namespace, runtime, p.log.Named("monitor-plugin-prometheus"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateStrategy(pluginName string, mp interfaces.Monitor) (interfaces.Strategy, error) {
	if pluginName == PluginStrategyTypeCanary {
		return canary.New(p.log.Named("strategy-plugin-canary"), mp)
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) CreateWebhook(pluginName string) (interfaces.Webhook, error) {
	if pluginName == PluginStrategyTypeDiscord {
		return discord.New(p.log.Named("webhook-plugin-discord"))
	}

	return nil, fmt.Errorf("not implemented")
}

func (p *ProviderImpl) GetLogger() hclog.Logger {
	return p.log
}

func (p *ProviderImpl) GetMetrics() interfaces.Metrics {
	return p.metrics
}

func (p *ProviderImpl) GetDataStore() interfaces.Store {
	return p.store
}

func (p *ProviderImpl) GetStateMachine(release *models.Release) (interfaces.StateMachine, error) {
	if r, ok := statemachines[release]; ok {
		return r, nil
	}

	sm, err := statemachine.New(release, p)
	if err != nil {
		return nil, fmt.Errorf("unable to create new statemachine: %s", err)
	}

	statemachines[release] = sm

	return sm, nil
}

func (p *ProviderImpl) DeleteStateMachine(release *models.Release) error {
	delete(statemachines, release)

	return nil
}
