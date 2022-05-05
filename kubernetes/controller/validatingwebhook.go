package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-v1-deployment,mutating=false,failurePolicy=fail,groups="",resources=deployments,verbs=create;update,versions=v1,name=controller-webhook.nicholasjackson.io,sideEffects=None,admissionReviewVersions=v1

// deploymentAdmission controls wether a new deployment is accepted by Kubernetes.
// Deployments should not be permitted when there is an active release.
type deploymentAdmission struct {
	Client   client.Client
	decoder  *admission.Decoder
	provider interfaces.Provider
	log      hclog.Logger
}

func NewDeploymentAdmission(client client.Client, provider interfaces.Provider, log hclog.Logger) *deploymentAdmission {
	return &deploymentAdmission{Client: client, provider: provider, log: log}
}

func (a *deploymentAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {
	deployment := &appsv1.Deployment{}

	err := a.decoder.Decode(req, deployment)
	if err != nil {
		a.log.Error("Error decoding deployment", "request", req, "error", err)

		return admission.Errored(http.StatusBadRequest, err)
	}

	a.log.Info("Handle deployment admission", "deployment", deployment.Name, "namespaces", deployment.Namespace, "labels", deployment.Labels)

	// was the deployment modified by the release controller, if so, ignore
	if deployment.Labels != nil &&
		deployment.Labels[interfaces.RuntimeDeploymentVersionLabel] != "" &&
		deployment.Labels[interfaces.RuntimeDeploymentVersionLabel] == deployment.ResourceVersion {

		a.log.Debug("Ignore deployment, resource was modified by the controller", "name", deployment.Name, "namespace", deployment.Namespace, "labels", deployment.Labels)

		return admission.Allowed("resource modified by controller")
	}

	// is there release for this deployment?
	rels, err := a.provider.GetDataStore().ListReleases(&interfaces.ListOptions{"kubernetes"})
	if err != nil {
		a.log.Error("Error fetching releases", "name", deployment.Name, "namespace", deployment.Namespace, "error", err)
		return admission.Errored(500, err)
	}

	for _, rel := range rels {
		conf := &kubernetes.PluginConfig{}
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

		a.log.Debug("Checking release", "name", deployment.Name, "namespace", deployment.Namespace, "regex", conf.DeploymentSelector)

		if re.MatchString(deployment.Name) && conf.Namespace == deployment.Namespace {
			// found a release for this deployment, check the state
			sm, err := a.provider.GetStateMachine(rel)
			if err != nil {
				a.log.Error("Error fetching statemachine", "name", deployment.Name, "namespace", deployment.Namespace, "error", err)
				return admission.Errored(500, err)
			}

			a.log.Debug("Found existing release for", "name", deployment.Name, "namespace", deployment.Namespace, "selector", conf.DeploymentSelector, "state", sm.CurrentState())

			if sm.CurrentState() == interfaces.StateIdle || sm.CurrentState() == interfaces.StateFail {
				// update the release with the absolute deployment name
				conf.CandidateName = deployment.Name
				confData, err := json.Marshal(conf)
				if err != nil {
					a.log.Error("Unable to serialize config", "conf", conf, "error", err)
					return admission.Errored(500, err)
				}

				rel.Runtime.Config = confData

				err = a.provider.GetDataStore().UpsertRelease(rel)
				if err != nil {
					a.log.Error("Unable to update the release", "release", rel, "error", err)
					return admission.Errored(500, err)
				}

				// clear any existing state
				a.provider.DeleteStateMachine(rel)

				// create a new statemachine
				sm, err := a.provider.GetStateMachine(rel)
				if err != nil {
					a.log.Error("Unable to get statemachine", "name", rel.Name, "error", err)
					return admission.Errored(500, err)
				}

				// kick off a new deployment
				err = sm.Deploy()
				if err != nil {
					a.log.Error("Error initializing new deployment", "name", deployment.Name, "namespace", deployment.Namespace, "error", err)
					return admission.Errored(500, err)
				}

				return admission.Allowed("New deployment created, initiating release")
			}

			// release currently active, reject deployment
			a.log.Debug("Reject deployment, there is currently an active release for this deployment", "name", deployment.Name, "namespace", deployment.Namespace, "state", sm.CurrentState())
			return admission.Denied("A release is currently active")
		}
	}

	return admission.Allowed("")
}

// podAnnotator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *deploymentAdmission) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
