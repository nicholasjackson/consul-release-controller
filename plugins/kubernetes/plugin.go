package kubernetes

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
)

type Plugin struct {
	log        hclog.Logger
	kubeClient interface{}
	config     *PluginConfig
}

func New(l hclog.Logger) *Plugin {
	return &Plugin{log: l}
}

type PluginConfig struct {
	Deployment string `hcl: "deployment" json:"deployment"`
}

func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}
	return json.Unmarshal(data, p.config)
}

func (p *Plugin) GetConfig() interface{} {
	return p.config
}

// Deploy the new test version to the platform
func (p *Plugin) Deploy(ctx context.Context, callback func()) error {
	// if this is a first deployment we need to clone the original deployment to create a primary

	// check the health of the new primary

	return nil
}

// Destroy removes any configuration that was created with the Deploy method
func (p *Plugin) Destroy(ctx context.Context, callback func()) error {
	return nil
}
