package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/sethvargo/go-retry"
)

type Plugin struct {
	log        hclog.Logger
	kubeClient clients.Kubernetes
	config     *PluginConfig
}

type PluginConfig struct {
	interfaces.RuntimeBaseConfig
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

func (p *Plugin) BaseConfig() interfaces.RuntimeBaseConfig {
	return p.config.RuntimeBaseConfig
}

// If primary deployment does not exist and the canary does (first run existing app)
//		copy the existing deployment and create a primary
//		wait until healthy
//		scale the traffic to the primary
//		monitor
// EventDeployed

// If primary deployment does not exist and the canary does not (first run no app)
//		copy the new deployment and create a primary
//		wait until healthy
//		scale the traffic to the primary
//		promote
// EventComplete

// If primary deployment exists (subsequent run existing app)
//		scale the traffic to the primary
//		monitor
// EventDeployed

// Deploy the new test version to the platform
func (p *Plugin) Deploy(ctx context.Context, status interfaces.RuntimeDeploymentStatus) error {
	p.log.Info("Setup Kubernetes deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	// we need to attempt this operation in a loop as the deployment might not have yet been created if this call comes from the
	// mutating webhook
	err := retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// grab a reference to the new deployment
		d, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
		if err != nil {
			p.log.Debug("Kubernetes deployment not found", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to find deployment: %s", err))
		}

		// have we already created the primary? if so return
		pd, err := p.kubeClient.GetDeployment(primaryName, p.config.Namespace)
		if err == nil && pd != nil {
			p.log.Debug("Kubernetes primary deployment already exists", "name", primaryName, "namespace", p.config.Namespace)
			return nil
		}

		// create a new primary appending primary to the deployment name
		p.log.Debug("Cloning deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)
		nd := d.DeepCopy()
		nd.Name = primaryName
		nd.ResourceVersion = ""

		// save the new primary
		err = p.kubeClient.UpsertDeployment(nd)
		if err != nil {
			p.log.Debug("Unable to upsert Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to clone deployment: %s", err))
		}

		p.log.Debug("Successfully cloned kubernetes deployment", "name", nd.Name, "namespace", nd.Namespace)

		return nil
	})

	if err != nil {
		return err
	}

	// check the health of the new primary
	err = p.checkDeploymentHealth(ctx, primaryName, p.config.Namespace)
	if err != nil {
		return err
	}

	// check the health of the new canary
	err = p.checkDeploymentHealth(ctx, p.config.Deployment, p.config.Namespace)
	if err != nil {
		return err
	}

	p.log.Info("Kubernetes deployment complete", "name", p.config.Deployment, "namespace", p.config.Namespace)
	return nil
}

func (p *Plugin) Promote(ctx context.Context) error {
	p.log.Info("Promote deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	// delete the primary and create a new primary from the canary
	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	// get the canary
	d, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
	if err != nil {
		p.log.Error("Unable to get canary", "name", p.config.Deployment, "namespace", p.config.Namespace)

		return err
	}

	// delete the old primary deployment
	p.log.Debug("Delete existing primary deployment", "name", primaryName, "namespace", p.config.Namespace)
	err = p.kubeClient.DeleteDeployment(primaryName, p.config.Namespace)
	if err != nil {
		p.log.Error("Unable to remove Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
		return fmt.Errorf("unable to remove previous primary deployment: %s", err)
	}

	err = retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		p.log.Debug("Checking deployment has been deleted", "name", primaryName, "namespace", p.config.Namespace)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		pd, err := p.kubeClient.GetDeployment(primaryName, p.config.Namespace)
		if err == nil {
			p.log.Debug("deployment still exists", "name", primaryName, "namespace", p.config.Namespace, "dep", pd)

			return retry.RetryableError(fmt.Errorf("deployment still exists"))
		}

		return nil
	})

	if err != nil {
		p.log.Error("Unable to remove Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
		return fmt.Errorf("unable to remove previous primary deployment: %s", err)
	}

	// create a new primary deployment from the canary
	p.log.Debug("Creating primary deployment from", "name", p.config.Deployment, "namespace", p.config.Namespace)
	nd := d.DeepCopy()
	nd.Name = primaryName
	nd.ResourceVersion = ""

	// save the new deployment
	err = p.kubeClient.UpsertDeployment(nd)
	if err != nil {
		p.log.Error("Unable to upsert Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
		return fmt.Errorf("unable to clone deployment: %s", err)
	}

	p.log.Debug("Successfully cloned kubernetes deployment", "name", primaryName, "namespace", nd.Namespace)

	err = p.checkDeploymentHealth(ctx, primaryName, d.Namespace)
	if err != nil {
		p.log.Error("Kubernetes deployment not healthy", "name", primaryName, "namespace", p.config.Namespace, "error", err)
		return fmt.Errorf("deployment not healthy: %s", err)
	}

	p.log.Info("Kubernetes promote complete", "name", p.config.Deployment, "namespace", p.config.Namespace)
	return nil
}

func (p *Plugin) Cleanup(ctx context.Context) error {
	p.log.Info("Cleanup old deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	// get the canary
	d, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
	if err != nil {
		p.log.Error("Unable to get canary", "name", p.config.Deployment, "namespace", p.config.Namespace)

		return err
	}

	// scale the canary to 0
	zero := int32(0)
	d.Spec.Replicas = &zero

	err = p.kubeClient.UpsertDeployment(d)
	if err != nil {
		p.log.Error("Unable to scale Kubernetes deployment", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)
		return err
	}

	return nil
}

// Destroy removes any configuration that was created with the Deploy method
func (p *Plugin) Destroy(ctx context.Context) error {

	return nil
}

func (p *Plugin) checkDeploymentHealth(ctx context.Context, name, namespace string) error {
	return retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		p.log.Info("Checking health", "name", name, "namespace", namespace)

		d, err := p.kubeClient.GetDeployment(name, namespace)
		if err != nil {
			p.log.Debug("Kubernetes deployment not found", "name", name, "namespace", namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to find deployment: %s", err))
		}

		p.log.Debug("Deployment health", "name", name, "namespace", namespace, "Status", d.Status)
		if d.Status.UnavailableReplicas > 0 || d.Status.AvailableReplicas < 1 {
			p.log.Debug("Kubernetes deployment not healthy", "name", name, "namespace", namespace)
			return retry.RetryableError(fmt.Errorf("deployment not healthy: %s", err))
		}

		p.log.Debug("Deployment healthy", "name", name, "namespace", namespace)
		return nil
	})
}
