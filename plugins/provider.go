package plugins

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/canary"
	"github.com/nicholasjackson/consul-release-controller/plugins/consul"
	"github.com/nicholasjackson/consul-release-controller/plugins/discord"
	"github.com/nicholasjackson/consul-release-controller/plugins/httptest"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-release-controller/plugins/prometheus"
	"github.com/nicholasjackson/consul-release-controller/plugins/slack"
	"github.com/nicholasjackson/consul-release-controller/plugins/statemachine"
)

var prov interfaces.Provider

// temporarily store the active statemachines here, at some point we need to
// think about serializing state
var statemachines map[string]interfaces.StateMachine

// GetProvider lazy instantiates a plugin provider and returns a reference
func GetProvider(log hclog.Logger, metrics interfaces.Metrics, store interfaces.Store) interfaces.Provider {
	if prov == nil {
		statemachines = map[string]interfaces.StateMachine{}
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
	return consul.New()
}

func (p *ProviderImpl) CreateRuntime(pluginName string) (interfaces.Runtime, error) {
	if pluginName == PluginRuntimeTypeKubernetes {
		return kubernetes.New()
	}

	return nil, fmt.Errorf("invalid Runtime plugin type: %s", pluginName)
}

func (p *ProviderImpl) CreateMonitor(pluginName, name, namespace, runtime string) (interfaces.Monitor, error) {
	if pluginName == PluginMonitorTypePromethus {
		return prometheus.New(name, namespace, runtime, p.log.Named("monitor-plugin-prometheus"))
	}

	return nil, fmt.Errorf("invalid Monitor plugin type: %s", pluginName)
}

func (p *ProviderImpl) CreateStrategy(pluginName string, mp interfaces.Monitor) (interfaces.Strategy, error) {
	if pluginName == PluginStrategyTypeCanary {
		return canary.New(mp)
	}

	return nil, fmt.Errorf("invalid Strategy plugin type: %s", pluginName)
}

func (p *ProviderImpl) CreateWebhook(pluginName string) (interfaces.Webhook, error) {
	switch pluginName {
	case PluginWebhookTypeDiscord:
		return discord.New()
	case PluginWebhookTypeSlack:
		return slack.New()
	}

	return nil, fmt.Errorf("invalid Webhook plugin type: %s", pluginName)
}

func (p *ProviderImpl) CreatePostDeploymentTest(pluginName, name, namespace, runtime string, mp interfaces.Monitor) (interfaces.PostDeploymentTest, error) {
	if pluginName == PluginDeploymentTestTypeHTTP {
		return httptest.New(name, namespace, runtime, mp)
	}

	return nil, fmt.Errorf("invalid Post deployment test plugin type: %s", pluginName)
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
	if r, ok := statemachines[getReleaseKey(release)]; ok {
		return r, nil
	}

	sm, err := statemachine.New(release, p)
	if err != nil {
		return nil, fmt.Errorf("unable to create new statemachine: %s", err)
	}

	statemachines[getReleaseKey(release)] = sm

	return sm, nil
}

func (p *ProviderImpl) DeleteStateMachine(release *models.Release) error {
	delete(statemachines, getReleaseKey(release))

	return nil
}

func getReleaseKey(release *models.Release) string {
	return fmt.Sprintf("%s-%s", release.Name, release.Namespace)
}
