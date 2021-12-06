package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/sethvargo/go-retry"
)

type Plugin struct {
	log        hclog.Logger
	kubeClient clients.Kubernetes
	config     *PluginConfig
}

type PluginConfig struct {
	Deployment string `hcl: "deployment" json:"deployment"`
	Namespace  string `hcl: "namespace" json:"namespace"`
}

func New(l hclog.Logger) (*Plugin, error) {
	kc, err := clients.NewKubernetes(os.Getenv("KUBECONFIG"))
	if err != nil {
		return nil, err
	}

	// create the client
	return &Plugin{log: l, kubeClient: kc}, nil
}

func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}
	return json.Unmarshal(data, p.config)
}

func (p *Plugin) GetConfig() interface{} {
	return p.config
}

// Deploy the new test version to the platform
func (p *Plugin) Deploy(ctx context.Context) error {
	// if this is a first deployment we need to clone the original deployment to create a primary
	p.log.Info("Creating kubernetes deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	// we need to attempt this operation in a loop as the deployment might not have yet been created if this call comes from the
	// mutating webhook
	err := retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		d, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
		if err != nil {
			p.log.Error("Kubernetes deployment not found", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to find deployment: %s", err))
		}

		// create a new deployment appending primary to the deployment name
		nd := d.DeepCopy()
		nd.Name = fmt.Sprintf("%s-primary", d.Name)
		nd.ResourceVersion = ""

		// save the new deployment
		err = p.kubeClient.UpsertDeployment(nd)
		if err != nil {
			p.log.Error("Unable to upsert Kubernetes deployment", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to clone deployment: %s", err))
		}

		p.log.Info("Cloned kubernetes deployment", "name", nd.Name, "namespace", nd.Namespace)

		return nil
	})

	// check the health of the new primary

	return err
}

// Destroy removes any configuration that was created with the Deploy method
func (p *Plugin) Destroy(ctx context.Context) error {
	return nil
}
