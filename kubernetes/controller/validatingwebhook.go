package controller

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/kr/pretty"
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
		return admission.Errored(http.StatusBadRequest, err)
	}

	// was the deployment modified by the release controller, if so, ignore
	pretty.Println(deployment.Labels)
	if deployment.Labels != nil && deployment.Labels["consul-release-controller-version"] == deployment.ResourceVersion {
		return admission.Allowed("resource modified by controller")
	}

	// is there release for this deployment?
	rels, err := a.provider.GetDataStore().ListReleases(&interfaces.ListOptions{"kubernetes"})
	for _, rel := range rels {
		conf := &kubernetes.PluginConfig{}
		json.Unmarshal(rel.Runtime.Config, conf)

		if conf.Deployment == deployment.Name {
			// found a release for this deployment, check the state
			sm, err := a.provider.GetStateMachine(rel)
			if err != nil {
				return admission.Errored(500, err)
			}

			a.log.Debug("Found existing release", "state", sm.CurrentState())

			if sm.CurrentState() == interfaces.StateIdle {
				// kick off a new deployment
				err = sm.Deploy()
				if err != nil {
					return admission.Errored(500, err)
				}

				return admission.Allowed("New deployment created, initiating release")
			}

			// release currently active, reject deployment
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
