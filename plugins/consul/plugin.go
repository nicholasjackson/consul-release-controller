package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/clients"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/sethvargo/go-retry"
)

var syncDelay = 2 * time.Second

type Plugin struct {
	log          hclog.Logger
	consulClient clients.Consul
	config       *PluginConfig
}

type PluginConfig struct {
	ConsulService string `json:"consul_service" validate:"required"`
}

var ErrConsulService = fmt.Errorf("ConsulService is a required field, please specify the name of the Consul service for the release.")

func New(l hclog.Logger) (*Plugin, error) {
	// create a new Consul client
	cc, err := clients.NewConsul()
	if err != nil {
		return nil, err
	}

	return &Plugin{log: l, consulClient: cc}, nil
}

func (s *Plugin) Configure(data json.RawMessage) error {
	s.config = &PluginConfig{}

	err := json.Unmarshal(data, s.config)

	// validate the plugin
	validate := validator.New()
	err = validate.Struct(s.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.ConsulService":
				errorMessage += ErrConsulService.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	return nil
}

// initialize is an internal function triggered by the initialize event
func (p *Plugin) Setup(ctx context.Context) error {
	p.log.Info("Initializing deployment", "service", p.config.ConsulService)

	// create the service defaults for the main service if they do not exist
	// TODO if the service defaults exist and they are not set to HTTP or gRPC we should fail as
	// we can not split traffic for TCP services
	p.log.Debug("Create service defaults", "service", p.config.ConsulService)
	err := p.consulClient.CreateServiceDefaults(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceDefaults", "name", p.config.ConsulService, "error", err)

		return err
	}

	// create the service resolver
	// creating the service resolver will interupt the application traffic temporarily
	p.log.Debug("Create service resolver", "service", p.config.ConsulService)
	err = p.consulClient.CreateServiceResolver(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceResolver", "name", p.config.ConsulService, "error", err)

		return err
	}

	// create the service router
	p.log.Debug("Create service router", "service", p.config.ConsulService)
	err = p.consulClient.CreateServiceRouter(p.config.ConsulService, false)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

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

	// update the router to remove the direct references to the canary and primary but
	// keep the retry, this is important to ensure no 503 errors as the clusters update
	p.log.Debug("Update router", "name", p.config.ConsulService)
	err := p.consulClient.CreateServiceRouter(p.config.ConsulService, true)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	p.log.Debug("Delete splitter", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceSplitter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceSplitter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	p.log.Debug("Delete resolver", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceResolver(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceResolver", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	p.log.Debug("Delete router", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceRouter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

		return err
	}

	time.Sleep(syncDelay)

	// delete will only happen if this plugin created the defaults
	p.log.Debug("Delete defaults", "name", p.config.ConsulService)
	err = p.consulClient.DeleteServiceDefaults(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to delete Consul ServiceDefaults", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) WaitUntilServiceHealthy(ctx context.Context, t interfaces.ServiceVariant) error {
	retryContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := retry.Constant(retryContext, 1*time.Second, func(ctx context.Context) error {
		p.log.Debug("Checking service is healthy", "name", p.config.ConsulService)

		err := p.consulClient.CheckHealth(p.config.ConsulService, t)
		if err != nil {
			p.log.Debug("Service not healthy, retrying", "name", p.config.ConsulService)
		}

		return nil
	})

	if err != nil {
		p.log.Error("Service health check failed", "service", p.config.ConsulService, "type", t, "error", err)
	}

	return err
}
