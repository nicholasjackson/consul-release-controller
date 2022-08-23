package nomad

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

var retryTimeout = 600 * time.Second
var retryInterval = 1 * time.Second

type Plugin struct {
	log         hclog.Logger
	store       interfaces.PluginStateStore
	config      *PluginConfig
	state       *PluginState
	nomadClient clients.Nomad
}

type PluginConfig struct {
	interfaces.RuntimeBaseConfig
}

type PluginState struct {
	interfaces.RuntimeBaseState
}

func New() (*Plugin, error) {

	// create the client
	return &Plugin{}, nil
}

func (p *Plugin) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	p.log = log
	p.store = store
	p.config = &PluginConfig{}

	err := json.Unmarshal(data, p.config)
	if err != nil {
		return err
	}

	if p.config.Namespace == "" {
		p.config.Namespace = "default"
	}

	// check to see if we have state that needs to be loaded
	p.state = &PluginState{}
	d, err := store.GetState()
	if err != nil {
		log.Debug("Unable to load state", "error", err)
	}

	err = json.Unmarshal(d, p.state)
	if err != nil {
		log.Debug("Unable to unmarshal state", "error", err)
	}

	p.nomadClient, err = clients.NewNomad(retryInterval, retryTimeout, log.ResetNamed("nomad-client"))
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) BaseConfig() interfaces.RuntimeBaseConfig {
	return p.config.RuntimeBaseConfig
}

func (p *Plugin) BaseState() interfaces.RuntimeBaseState {
	return p.state.RuntimeBaseState
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
func (p *Plugin) InitPrimary(ctx context.Context, releaseName string) (interfaces.RuntimeDeploymentStatus, error) {
	p.log.Info("Init the Primary deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)

	// save the state on exit
	defer p.saveState()

	p.state.PrimaryName = fmt.Sprintf("%s-primary", releaseName)

	var primaryDeployment *api.Job
	var candidateDeployment *api.Job
	var err error

	// have we already created the primary? if so return
	_, primaryErr := p.nomadClient.GetJob(ctx, p.state.PrimaryName, p.config.Namespace)

	// if we already have a primary exit
	if primaryErr == nil {
		p.log.Debug("Primary deployment already exists", "name", p.state.PrimaryName, "namespace", p.config.Namespace)

		return interfaces.RuntimeDeploymentNoAction, nil
	}

	// fetch the current deployment
	candidateDeployment, err = p.nomadClient.GetJobWithSelector(ctx, p.config.DeploymentSelector, p.config.Namespace)
	// if we have no Candidate there is nothing we can do
	if err != nil || candidateDeployment == nil {
		p.log.Debug("No candidate deployment, nothing to do")
		return interfaces.RuntimeDeploymentNoAction, nil
	}

	// update the internal name that will be used later
	p.state.CandidateName = *candidateDeployment.Name

	// create a new primary appending primary to the deployment name
	p.log.Debug("Cloning deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)
	primaryDeployment = candidateDeployment
	primaryDeployment.Name = &p.state.PrimaryName
	primaryDeployment.ID = &p.state.PrimaryName
	primaryDeployment.SetMeta("ResourceVersion", "primary")

	primaryDeployment.Version = nil
	primaryDeployment.JobModifyIndex = nil

	// add tags to the consul service
	for _, tg := range primaryDeployment.TaskGroups {
		for _, s := range tg.Services {
			if s.Tags == nil {
				s.Tags = []string{}
			}

			s.Tags = append(s.Tags, "primary")
		}
	}

	//// save the new primary
	err = p.nomadClient.UpsertJob(ctx, primaryDeployment)
	if err != nil {
		p.log.Debug("Unable to create Primary deployment", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to clone deployment: %s", err)
	}

	// check the health of the primary
	_, err = p.nomadClient.GetHealthyJob(ctx, p.state.PrimaryName, p.config.Namespace)
	if err != nil {
		return interfaces.RuntimeDeploymentInternalError, err
	}

	p.log.Debug("Successfully cloned Nomad job", "name", *primaryDeployment.Name, "namespace", *primaryDeployment.Namespace)

	p.log.Info("Init primary complete", "candidate", p.state.CandidateName, "primary", p.state.PrimaryName, "namespace", p.config.Namespace)
	return interfaces.RuntimeDeploymentUpdate, nil
}

func (p *Plugin) PromoteCandidate(ctx context.Context) (interfaces.RuntimeDeploymentStatus, error) {
	p.log.Info("Promote deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)

	// save the state on exit
	defer p.saveState()

	// the deployment might not yet exist due to eventual consistency
	candidate, err := p.nomadClient.GetHealthyJob(ctx, p.state.CandidateName, p.config.Namespace)

	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Candidate job does not exist", "name", p.state.CandidateName, "namespace", p.config.Namespace)

		return interfaces.RuntimeDeploymentNotFound, nil
	}

	if err != nil {
		p.log.Error("Unable to get Candidate", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, err
	}

	// delete the old primary deployment if exists
	p.log.Debug("Delete existing Primary job", "name", p.state.PrimaryName, "namespace", p.config.Namespace)
	err = p.nomadClient.DeleteJob(ctx, p.state.PrimaryName, p.config.Namespace)
	if err != nil {
		p.log.Error("Unable to remove Nomad job", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err)
		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to remove previous primary deployment: %s", err)
	}

	// create a new primary deployment from the canary
	p.log.Debug("Creating Primary job from", "name", p.state.CandidateName, "namespace", p.config.Namespace)
	primary := candidate
	primary.Name = &p.state.PrimaryName
	primary.ID = &p.state.PrimaryName
	primary.SetMeta("ResourceVersion", "primary")
	primary.Version = nil

	// add tags to the consul service
	for _, tg := range primary.TaskGroups {
		for _, s := range tg.Services {
			if s.Tags == nil {
				s.Tags = []string{}
			}

			s.Tags = append(s.Tags, "primary")
		}
	}

	// save the new deployment
	err = p.nomadClient.UpsertJob(ctx, primary)
	if err != nil {
		p.log.Error("Unable to create Primary deployment", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "dep", primary, "error", err)

		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("unable to clone deployment: %s", err)
	}

	p.log.Debug("Successfully created new Primary deployment", "name", p.state.PrimaryName, "namespace", primary.Namespace)

	// wait for deployment healthy
	_, err = p.nomadClient.GetHealthyJob(ctx, p.state.PrimaryName, p.config.Namespace)
	if err != nil {
		p.log.Error("Primary deployment not healthy", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err)

		return interfaces.RuntimeDeploymentInternalError, fmt.Errorf("deployment not healthy: %s", err)
	}

	p.log.Info("Promote complete", "candidate", p.state.CandidateName, "primary", p.state.PrimaryName, "namespace", p.config.Namespace)

	return interfaces.RuntimeDeploymentUpdate, nil
}

func (p *Plugin) RemoveCandidate(ctx context.Context) error {
	p.log.Info("Remove candidate deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)

	// save the state on exit
	defer p.saveState()

	// get the candidate
	d, err := p.nomadClient.GetJob(ctx, p.state.CandidateName, p.config.Namespace)
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Candidate not found", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)

		return nil
	}

	if err != nil {
		p.log.Error("Unable to get candidate", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)

		return err
	}

	// scale the canary to 0
	zero := int(0)
	for _, tg := range d.TaskGroups {
		tg.Count = &zero
	}

	err = p.nomadClient.UpsertJob(ctx, d)
	if err != nil {
		p.log.Error("Unable to scale Kubernetes deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)
		return err
	}

	return nil
}

// RestoreOriginal restores the original deployment cloned to the primary
// because the last deployed version might be different from the primary due to a rollback
// we will copy the primary to the canary, rather than scale it up
func (p *Plugin) RestoreOriginal(ctx context.Context) error {
	p.log.Info("Restore original deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)

	// save the state on exit
	defer p.saveState()

	// get the primary
	primaryDeployment, err := p.nomadClient.GetJob(ctx, p.state.PrimaryName, p.config.Namespace)

	// if there is no primary, return, nothing we can do
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Primary does not exist, exiting", "name", p.state.PrimaryName, "namespace", p.config.Namespace)

		return nil
	}

	// if a general error is returned from the get operation, fail
	if err != nil {
		p.log.Error("Unable to get primary", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err, "dep", primaryDeployment)

		return err
	}

	p.log.Debug("Delete existing candidate deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace)

	err = p.nomadClient.DeleteJob(ctx, p.state.CandidateName, p.config.Namespace)
	if err != nil && err != clients.ErrDeploymentNotFound {
		p.log.Error("Unable to remove existing candidate deployment", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err)

		return fmt.Errorf("unable to remove previous candidate deployment: %s", err)
	}

	// create canary from the current primary
	cd := primaryDeployment
	cd.ID = &p.state.CandidateName
	cd.Name = &p.state.CandidateName
	cd.Version = nil

	// remove the primary tag if set
	for _, tg := range cd.TaskGroups {
		for _, s := range tg.Services {
			tags := []string{}
			for _, t := range s.Tags {
				if t != "primary" {
					tags = append(tags, t)
				}
			}

			s.Tags = tags
		}
	}

	// remove the ownership label so that it can be updated as normal
	//delete(cd.Labels, interfaces.RuntimeDeploymentVersionLabel)

	p.log.Debug("Clone primary to create original deployment", "primary", p.state.PrimaryName, "candidate", p.state.CandidateName, "namespace", p.config.Namespace)

	err = p.nomadClient.UpsertJob(ctx, cd)
	if err != nil {
		p.log.Error("Unable to restore original deployment", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)

		return err
	}

	// wait for health checks
	_, err = p.nomadClient.GetHealthyJob(ctx, *cd.Name, *cd.Namespace)
	if err != nil {
		p.log.Error("Original deployment not healthy", "name", p.state.CandidateName, "namespace", p.config.Namespace, "error", err)

		return err
	}

	return nil
}

func (p *Plugin) RemovePrimary(ctx context.Context) error {
	p.log.Info("Remove primary deployment", "name", p.state.PrimaryName, "namespace", p.config.Namespace)

	// save the state on exit
	defer p.saveState()

	//// delete the primary
	err := p.nomadClient.DeleteJob(ctx, p.state.PrimaryName, p.config.Namespace)

	// if there is no primary, return, nothing we can do
	if err == clients.ErrDeploymentNotFound {
		p.log.Debug("Primary does not exist, exiting", "name", p.state.PrimaryName, "namespace", p.config.Namespace)

		return nil
	}

	if err != nil {
		p.log.Error("Unable to delete primary", "name", p.state.PrimaryName, "namespace", p.config.Namespace, "error", err)

		return err
	}

	return nil
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
func (p *Plugin) CandidateSubsetFilter() string {
	return fmt.Sprintf(`"primary" not in Service.Tags`)
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
func (p *Plugin) PrimarySubsetFilter() string {
	return fmt.Sprintf(`"primary" in Service.Tags`)
}

func (p *Plugin) saveState() {
	d, err := json.Marshal(p.state)
	if err != nil {
		p.log.Error("Unable to marshal state to json", "error", err)
		return
	}

	err = p.store.UpsertState(d)
	if err != nil {
		p.log.Error("Unable to save state", "error", err)
	}
}
