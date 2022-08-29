package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/sethvargo/go-retry"
)

var syncDelay = 1 * time.Second

type Plugin struct {
	log          hclog.Logger
	store        interfaces.PluginStateStore
	consulClient clients.Consul
	config       *PluginConfig
}

type PluginConfig struct {
	interfaces.ReleaserBaseConfig
}

var ErrConsulService = fmt.Errorf("ConsulService is a required field, please specify the name of the Consul service for the release.")

func New() (*Plugin, error) {
	return &Plugin{}, nil
}

func (s *Plugin) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	s.log = log
	s.store = store
	s.config = &PluginConfig{}

	err := json.Unmarshal(data, s.config)

	// validate the plugin
	validate := validator.New()
	err = validate.Struct(s.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.ReleaserBaseConfig.ConsulService":
				errorMessage += ErrConsulService.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	opts := &clients.ConsulOptions{
		Namespace: s.config.Namespace,
		Partition: s.config.Partition,
	}

	// create a new Consul client
	cc, err := clients.NewConsul(opts)
	if err != nil {
		return err
	}

	s.consulClient = cc

	s.log.Debug("Configured Consul Releaser plugin", "service", s.config.ConsulService, "namespace", s.config.Namespace, "partition", s.config.Partition)

	return nil
}

func (s *Plugin) BaseConfig() interfaces.ReleaserBaseConfig {
	return s.config.ReleaserBaseConfig
}

// initialize is an internal function triggered by the initialize event
func (p *Plugin) Setup(ctx context.Context, primarySubsetFilter, candidateSubsetFilter string) error {
	p.log.Info("Initializing deployment", "service", p.config.ConsulService)

	// create the service defaults for the main service if they do not exist
	// If the service defaults exist and they are not set to HTTP we will fail as we
	// should not overwite
	p.log.Debug("Create service defaults", "service", p.config.ConsulService)
	err := p.consulClient.CreateServiceDefaults(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceDefaults", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// create the service defaults for the controller and the virtual service that allows
	// access to candidate deployments
	err = p.consulClient.CreateServiceDefaults(clients.ControllerServiceName)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceDefaults", "name", clients.ControllerServiceName, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	err = p.consulClient.CreateServiceDefaults(clients.UpstreamRouterName)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceDefaults", "name", clients.UpstreamRouterName, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// create the service resolver
	p.log.Debug("Create service resolver", "service", p.config.ConsulService)
	err = p.consulClient.CreateServiceResolver(p.config.ConsulService, primarySubsetFilter, candidateSubsetFilter)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceResolver", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// create the service router to enable post deployment tests
	p.log.Debug("Create upstream service router", "service", p.config.ConsulService)
	err = p.consulClient.CreateUpstreamRouter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// create the service intentions to allow an upstream from the controller to
	p.log.Debug("Create service intentions for the upstreams", "service", p.config.ConsulService)
	err = p.consulClient.CreateServiceIntention(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceIntention", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) Scale(ctx context.Context, value int) error {
	primaryTraffic := 100 - value
	canaryTraffic := value

	p.log.Info("Scale deployment", "name", p.config.ConsulService, "traffic_primary", primaryTraffic, "traffic_canary", canaryTraffic)

	// create the service spiltter set to 100% primary
	err := p.consulClient.CreateServiceSplitter(p.config.ConsulService, primaryTraffic, canaryTraffic)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceSplitter", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) Destroy(ctx context.Context) error {
	p.log.Info("Remove Consul config", "name", p.config.ConsulService)

	p.log.Debug("Delete splitter", "name", p.config.ConsulService)
	err := p.consulClient.DeleteServiceSplitter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceSplitter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	p.log.Debug("Cleanup upstream router", "name", p.config.ConsulService)
	err = p.consulClient.DeleteUpstreamRouter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete upstream Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	p.log.Debug("Cleanup resolver", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceResolver(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceResolver", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// delete will only happen if this plugin created the defaults
	p.log.Debug("Cleanup service intentions", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceIntention(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceIntention", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// delete will only happen if this plugin created the defaults
	p.log.Debug("Cleanup defaults", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceDefaults(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceDefaults", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) WaitUntilServiceHealthy(ctx context.Context, filter string) error {
	retryContext, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	err := retry.Constant(retryContext, 1*time.Second, func(ctx context.Context) error {
		p.log.Debug("Checking service is healthy", "name", p.config.ConsulService)

		err := p.consulClient.CheckHealth(p.config.ConsulService, filter)
		if err != nil {
			p.log.Debug("Service not healthy, retrying", "name", p.config.ConsulService)
			return retry.RetryableError(err)
		}

		return nil
	})

	if err != nil {
		p.log.Error("Service health check failed", "service", p.config.ConsulService, "filter", filter, "error", err)
	}

	return err
}
