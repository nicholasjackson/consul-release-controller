package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type AdmissionResponse string

const (
	AdmissionError    AdmissionResponse = "admission_error"
	AdmissionGranted  AdmissionResponse = "admission_granted"
	AdmissionRejected AdmissionResponse = "admission_rejected"
)

type Admission interface {
	// Check if the given release is allowed to be admitted by the system
	// If the current state of the release that matches the deployment criteria is not state_idle,
	// AdmissionRejected with be returned along with an error that explains the reason for the rejection
	// In the instance of an internal error AdmissionError is returned along with the error message
	// If the Admission is successful AdmissionGranted is returned with a nil error
	Check(ctx context.Context, name string, namespace string, labels map[string]string, version string, runtime string) (AdmissionResponse, error)
}

type AdmissionImpl struct {
	provider interfaces.Provider
	log      hclog.Logger
}

func NewAdmission(p interfaces.Provider, l hclog.Logger) Admission {
	return &AdmissionImpl{p, l}
}

func (a *AdmissionImpl) Check(ctx context.Context, name string, namespace string, labels map[string]string, version string, runtime string) (AdmissionResponse, error) {
	a.log.Info("Handle deployment admission", "deployment", name, "namespaces", namespace, "labels", labels)

	// was the deployment modified by the release controller, if so, ignore
	if labels != nil &&
		labels[interfaces.RuntimeDeploymentVersionLabel] != "" &&
		labels[interfaces.RuntimeDeploymentVersionLabel] == version {

		a.log.Debug("Ignore deployment, resource was modified by the controller", "name", name, "namespace", namespace, "labels", labels, "version", version)

		return AdmissionGranted, nil
	}

	// is there release for this deployment?
	rels, err := a.provider.GetDataStore().ListReleases(&interfaces.ListOptions{Runtime: runtime})
	if err != nil {
		a.log.Error("Error fetching releases", "name", name, "namespace", namespace, "error", err)
		return AdmissionError, err
	}

	for _, rel := range rels {
		conf := &interfaces.RuntimeBaseConfig{}
		json.Unmarshal(rel.Runtime.Config, conf)

		// PluginConfig.Deployment can reference deployments using regular expressions
		// check if this matches

		//first check to see if the regex terminates in $ (word boundary), if not add it
		if !strings.HasSuffix(conf.DeploymentSelector, "$") {
			conf.DeploymentSelector = conf.DeploymentSelector + "$"
		}

		re, err := regexp.Compile(conf.DeploymentSelector)
		if err != nil {
			a.log.Error("Invalid regular expression for deployment in release config", "release", rel.Name, "error", err)
			continue
		}

		a.log.Debug("Checking release", "name", name, "namespace", namespace, "regex", conf.DeploymentSelector)

		if re.MatchString(name) && conf.Namespace == namespace {
			// found a release for this deployment, check the state
			sm, err := a.provider.GetStateMachine(rel)
			if err != nil {
				a.log.Error("Error fetching state machine", "name", name, "namespace", namespace, "error", err)
				return AdmissionError, err
			}

			a.log.Debug("Found existing release for", "name", name, "namespace", namespace, "selector", conf.DeploymentSelector, "state", sm.CurrentState())

			if sm.CurrentState() == interfaces.StateDestroy {
				a.log.Debug("Ignoring release, destroy state", "name", rel.Name)
				return AdmissionGranted, nil
			}

			// if the state of the release is inactive, update the config
			if sm.CurrentState() == interfaces.StateIdle || sm.CurrentState() == interfaces.StateFail {

				// update the release candidate name so that the runtime plugin knows which deployment to clone
				a.log.Debug("Fetch plugin state", "name", rel.Name)
				ds := a.provider.GetDataStore().CreatePluginStateStore(rel, "runtime")

				ps := &interfaces.RuntimeBaseState{}
				d, err := ds.GetState()
				if err != nil {
					a.log.Error("Unable to fetch state", "name", rel.Name, "error", err)
				}

				err = json.Unmarshal(d, ps)
				if err != nil {
					a.log.Error("Unable to unmarshal state", "name", rel.Name, "error", err)
				}

				// update the candidate name
				// TODO, find a better way of updating the state than this
				a.log.Debug("Set CandidateName to plugin state", "name", rel.Name, "candidate_name", name)
				ps.CandidateName = name

				confData, err := json.Marshal(ps)
				if err != nil {
					a.log.Error("Unable to serialize config", "conf", conf, "error", err)
					return AdmissionError, err
				}

				err = ds.UpsertState(confData)
				if err != nil {
					a.log.Error("Unable to save runtime plugin state", "conf", conf, "error", err)
					return AdmissionError, err
				}

				// clear any existing state
				a.provider.DeleteStateMachine(rel)

				// create a new statemachine
				sm, err := a.provider.GetStateMachine(rel)
				if err != nil {
					a.log.Error("Unable to get statemachine", "name", rel.Name, "error", err)
					return AdmissionError, err
				}

				// kick off a new deployment
				err = sm.Deploy()
				if err != nil {
					a.log.Error("Error initializing new deployment", "name", name, "namespace", namespace, "error", err)
					return AdmissionError, err
				}

				return AdmissionGranted, nil
			}

			// release currently active, reject deployment
			a.log.Debug("Reject deployment, there is currently an active release for this deployment", "name", name, "namespace", namespace, "state", sm.CurrentState())
			return AdmissionRejected, fmt.Errorf("A release for the deployment %s currently active, state: %s", name, sm.CurrentState())
		}
	}

	// no matching release allow entry
	return AdmissionGranted, nil
}
