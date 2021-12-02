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

func New(l hclog.Logger) *Plugin {
	// create a new Consul client
	cc := &clients.MockConsul{}
	return &Plugin{log: l, consulClient: cc}
}

func (s *Plugin) Configure(data json.RawMessage) error {
	s.config = &PluginConfig{}
	return json.Unmarshal(data, s.config)
}

// initialize is an internal function triggered by the initialize event
func (p *Plugin) Setup(ctx context.Context, done func()) error {
	p.log.Debug("initializing deployment", "service", p.config.ConsulService)

	// create the service defaults for the primary and the canary
	err := p.consulClient.CreateServiceDefaults(fmt.Sprintf("cc-%s-primary", p.config.ConsulService))
	if err != nil {
		return err
	}

	err = p.consulClient.CreateServiceDefaults(fmt.Sprintf("cc-%s-canary", p.config.ConsulService))
	if err != nil {
		return err
	}

	// create the service resolver
	err = p.consulClient.CreateServiceResolver(fmt.Sprintf("cc-%s", p.config.ConsulService))
	if err != nil {
		return err
	}

	// create the service router
	err = p.consulClient.CreateServiceRouter(fmt.Sprintf("cc-%s", p.config.ConsulService))
	if err != nil {
		return err
	}

	// create the service spiltter set to 100% primary
	err = p.consulClient.CreateServiceSplitter(fmt.Sprintf("cc-%s", p.config.ConsulService), 100, 0)
	if err != nil {
		return err
	}

	// call the callback
	done()

	return nil
}

func (s *Plugin) Destroy(ctx context.Context, done func()) error {
	return fmt.Errorf("not implemented")
}

func (s *Plugin) Scale(ctx context.Context, value int, done func()) error {
	return fmt.Errorf("not implemented")
}
