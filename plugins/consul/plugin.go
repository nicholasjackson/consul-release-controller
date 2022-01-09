package consul

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
)

type Plugin struct {
	log          hclog.Logger
	consulClient clients.Consul
	config       *PluginConfig
}

type PluginConfig struct {
	ConsulService string `json:"consul_service"`
}

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
	return json.Unmarshal(data, s.config)
}

// initialize is an internal function triggered by the initialize event
func (p *Plugin) Setup(ctx context.Context) error {
	p.log.Info("initializing deployment", "service", p.config.ConsulService)

	// create the service defaults for the main service
	err := p.consulClient.CreateServiceDefaults(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceDefaults", "name", p.config.ConsulService, "error", err)

		return err
	}

	// create the service resolver
	err = p.consulClient.CreateServiceResolver(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceResolver", "name", p.config.ConsulService, "error", err)

		return err
	}

	// create the service spiltter set to 100% primary
	err = p.consulClient.CreateServiceSplitter(p.config.ConsulService, 100, 0)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceSplitter", "name", p.config.ConsulService, "error", err)

		return err
	}

	// create the service router
	err = p.consulClient.CreateServiceRouter(p.config.ConsulService)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceRouter", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) Destroy(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (p *Plugin) Scale(ctx context.Context, value int) error {
	primaryTraffic := 100 - value
	canaryTraffic := value

	p.log.Info("scale deployment", "name", p.config.ConsulService, "traffic_primary", primaryTraffic, "traffic_canary", canaryTraffic)

	// create the service spiltter set to 100% primary
	err := p.consulClient.CreateServiceSplitter(p.config.ConsulService, primaryTraffic, canaryTraffic)
	if err != nil {
		p.log.Error("Unable to create Consul ServiceSplitter", "name", p.config.ConsulService, "error", err)

		return err
	}

	return nil
}
