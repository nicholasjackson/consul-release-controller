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
	v1 "k8s.io/api/apps/v1"
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

// InitPrimary creates a primary from the new deployment, if the primary already exists do nothing
// to replace the primary call PromoteCanary
func (p *Plugin) InitPrimary(ctx context.Context) (interfaces.RuntimeDeploymentStatus, error) {
	p.log.Info("Init the Primary deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	var primaryDeployment *v1.Deployment
	var canaryDeployment *v1.Deployment
	var err error

	// have we already created the primary? if so return
	_, err = p.kubeClient.GetDeployment(primaryName, p.config.Namespace)
	if err == nil {
		p.log.Debug("Kubernetes primary deployment already exists", "name", primaryName, "namespace", p.config.Namespace)

		return interfaces.RuntimeDeploymentNoAction, nil
	}

	// we need to attempt this operation in a loop as the deployment might not have yet been created if this call comes from the
	// mutating webhook
	retryContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = retry.Fibonacci(retryContext, 1*time.Second, func(ctx context.Context) error {
		// if the context has timed out or been cancelled, return
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// grab a reference to the new deployment
		var getErr error
		canaryDeployment, getErr = p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
		if getErr != nil {
			p.log.Debug("Canary deployment not found", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", getErr)

			return retry.RetryableError(fmt.Errorf("unable to find deployment: %s", err))
		}

		return nil
	})

	// if we have no Canary there is nothing we can do
	if err != nil {
		return interfaces.RuntimeDeploymentNoAction, nil
	}

	// create a new primary appending primary to the deployment name
	p.log.Debug("Cloning deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)
	primaryDeployment = canaryDeployment.DeepCopy()
	primaryDeployment.Name = primaryName
	primaryDeployment.ResourceVersion = ""

	// save the new primary
	err = p.kubeClient.UpsertDeployment(primaryDeployment)
	if err != nil {
		p.log.Debug("Unable to upsert Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to clone deployment: %s", err)
	}

	// check the health of the primary
	err = p.checkDeploymentHealth(ctx, primaryName, p.config.Namespace)
	if err != nil {
		return interfaces.RuntimeDeploymentInternalError, err
	}

	p.log.Debug("Successfully cloned kubernetes deployment", "name", primaryDeployment.Name, "namespace", primaryDeployment.Namespace)

	p.log.Info("Init primary complete", "name", p.config.Deployment, "namespace", p.config.Namespace)
	return interfaces.RuntimeDeploymentUpdate, nil
}

func (p *Plugin) PromoteCanary(ctx context.Context) (interfaces.RuntimeDeploymentStatus, error) {
	p.log.Info("Promote deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	// delete the primary and create a new primary from the canary
	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	var canary *v1.Deployment

	// the deployment might not yet exist due to eventual consistency
	retryContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := retry.Fibonacci(retryContext, 1*time.Second, func(ctx context.Context) error {
		var err error
		canary, err = p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
		if err != nil {
			return retry.RetryableError(err)
		}

		return nil
	})

	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Canary deployment does not exist", "name", p.config.Deployment, "namespace", p.config.Namespace)

		return interfaces.RuntimeDeploymentNotFound, nil
	}

	if err != nil {
		p.log.Error("Unable to get canary", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, err
	}

	// delete the old primary deployment if exists
	_, err = p.kubeClient.GetDeployment(primaryName, p.config.Namespace)
	if err != nil && err != clients.ErrDeploymentNotFound {
		p.log.Error("Unable to get details for primary deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, err
	}

	// we have a primary so delete it
	if err == nil {
		p.log.Debug("Delete existing primary deployment", "name", primaryName, "namespace", p.config.Namespace)
		err = p.kubeClient.DeleteDeployment(primaryName, p.config.Namespace)
		if err != nil {
			p.log.Error("Unable to remove Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
			return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to remove previous primary deployment: %s", err)
		}

		retryContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		err = retry.Fibonacci(retryContext, 1*time.Second, func(ctx context.Context) error {
			p.log.Debug("Checking deployment has been deleted", "name", primaryName, "namespace", p.config.Namespace)

			if ctx.Err() != nil {
				return ctx.Err()
			}

			pd, err := p.kubeClient.GetDeployment(primaryName, p.config.Namespace)
			if err == nil || err != clients.ErrDeploymentNotFound {
				p.log.Debug("deployment still exists", "name", primaryName, "namespace", p.config.Namespace, "dep", pd)

				return retry.RetryableError(fmt.Errorf("deployment still exists"))
			}

			if err == clients.ErrDeploymentNotFound {
				return nil
			}

			return nil
		})

		if err != nil {
			p.log.Error("Unable to remove Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)
			return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to remove previous primary deployment: %s", err)
		}
	}

	// create a new primary deployment from the canary
	p.log.Debug("Creating primary deployment from", "name", p.config.Deployment, "namespace", p.config.Namespace)
	nd := canary.DeepCopy()
	nd.Name = primaryName
	nd.ResourceVersion = ""

	// save the new deployment
	err = p.kubeClient.UpsertDeployment(nd)
	if err != nil {
		p.log.Error("Unable to upsert Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "dep", nd, "error", err)

		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to clone deployment: %s", err)
	}

	p.log.Debug("Successfully cloned kubernetes deployment", "name", primaryName, "namespace", nd.Namespace)

	err = p.checkDeploymentHealth(ctx, primaryName, canary.Namespace)
	if err != nil {
		p.log.Error("Kubernetes deployment not healthy", "name", primaryName, "namespace", p.config.Namespace, "error", err)
		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("deployment not healthy: %s", err)
	}

	p.log.Info("Kubernetes promote complete", "name", p.config.Deployment, "namespace", p.config.Namespace)

	return interfaces.RuntimeDeploymentUpdate, nil
}

func (p *Plugin) RemoveCanary(ctx context.Context) error {
	p.log.Info("Cleanup old deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	// get the canary
	d, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Canary not found", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)

		return nil
	}

	if err != nil {
		p.log.Error("Unable to get canary", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)

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

// RestoreOriginal restores the original deployment cloned to the primary
// because the last deployed version might be different from the primary due to a rollback
// we will copy the primary to the canary, rather than scale it up
func (p *Plugin) RestoreCanary(ctx context.Context) error {
	p.log.Info("Restore original deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	// get the primary
	primaryDeployment, err := p.kubeClient.GetDeployment(primaryName, p.config.Namespace)

	// if there is no primary, return, nothing we can do
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Primary does not exist, exiting", "name", primaryName, "namespace", p.config.Namespace)

		return nil
	}

	// if a general error is returned from the get operation, fail
	if err != nil {
		p.log.Error("Unable to get primary", "name", primaryName, "namespace", p.config.Namespace, "error", err, "dep", primaryDeployment)

		return err
	}

	p.log.Debug("Delete existing canary deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	err = p.kubeClient.DeleteDeployment(p.config.Deployment, p.config.Namespace)
	if err != nil && err != clients.ErrDeploymentNotFound {
		p.log.Error("Unable to remove existing canary deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)

		return fmt.Errorf("unable to remove previous canary deployment: %s", err)
	}

	// check the deployment has been fully removed
	retryContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = retry.Fibonacci(retryContext, 1*time.Second, func(ctx context.Context) error {
		p.log.Debug("Checking canary has been deleted", "name", p.config.Deployment, "namespace", p.config.Namespace)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := p.kubeClient.GetDeployment(p.config.Deployment, p.config.Namespace)
		if err == clients.ErrDeploymentNotFound {
			return nil
		}

		p.log.Debug("Deployment still exists", "name", p.config.Deployment, "namespace", p.config.Namespace)

		return retry.RetryableError(fmt.Errorf("deployment still exists"))
	})

	if err != nil {
		p.log.Error("Unable to remove Kubernetes deployment", "name", primaryName, "namespace", p.config.Namespace, "error", err)

		return fmt.Errorf("unable to remove previous canary deployment: %s", err)
	}

	// create canary from the current primary
	cd := primaryDeployment.DeepCopy()
	cd.Name = p.config.Deployment
	cd.ResourceVersion = ""

	p.log.Debug("Clone primary to create canary deployment", "name", p.config.Deployment, "namespace", p.config.Namespace, "dep", cd)

	err = p.kubeClient.UpsertDeployment(cd)
	if err != nil {
		p.log.Error("Unable to create new canary deployment", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)

		return err
	}

	// wait for health checks
	err = p.checkDeploymentHealth(ctx, cd.Name, cd.Namespace)
	if err != nil {
		p.log.Error("Canary deployment not healthy", "name", p.config.Deployment, "namespace", p.config.Namespace, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) RemovePrimary(ctx context.Context) error {
	p.log.Info("Remove primary deployment", "name", p.config.Deployment, "namespace", p.config.Namespace)

	primaryName := fmt.Sprintf("%s-primary", p.config.Deployment)

	// get the primary
	pd, err := p.kubeClient.GetDeployment(primaryName, p.config.Namespace)

	// if a general error is returned from the get operation, fail
	if err != nil && err != clients.ErrDeploymentNotFound {
		p.log.Error("Unable to get primary", "name", primaryName, "namespace", p.config.Namespace, "error", err, "dep", pd)

		return err
	}

	// if there is no primary, return, nothing we can do
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Primary does not exist, exiting", "name", primaryName, "namespace", p.config.Namespace)
		return nil
	}

	// delete the primary
	err = p.kubeClient.DeleteDeployment(primaryName, pd.Namespace)
	if err != nil {
		p.log.Error("Unable to delete primary", "name", pd.Name, "namespace", pd.Namespace, "error", err)
		return err
	}

	return nil
}

func (p *Plugin) checkDeploymentHealth(ctx context.Context, name, namespace string) error {
	return retry.Fibonacci(ctx, 1*time.Second, func(ctx context.Context) error {
		p.log.Debug("Checking health", "name", name, "namespace", namespace)

		d, err := p.kubeClient.GetDeployment(name, namespace)
		if err != nil {
			p.log.Debug("Kubernetes deployment not found", "name", name, "namespace", namespace, "error", err)
			return retry.RetryableError(fmt.Errorf("unable to find deployment: %s", err))
		}

		p.log.Debug("Deployment health", "name", name, "namespace", namespace, "status_replicas", d.Status.AvailableReplicas, "desired_replicas", d.Status.Replicas)

		if d.Status.UnavailableReplicas > 0 || d.Status.AvailableReplicas < 1 {
			p.log.Debug("Kubernetes deployment not healthy", "name", name, "namespace", namespace)
			return retry.RetryableError(fmt.Errorf("deployment not healthy: %s", err))
		}

		p.log.Debug("Deployment healthy", "name", name, "namespace", namespace)
		return nil
	})
}
