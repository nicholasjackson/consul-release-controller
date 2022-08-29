package controller

import (
	"context"
	"net/http"

	admissionController "github.com/nicholasjackson/consul-release-controller/pkg/controllers"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-v1-deployment,mutating=false,failurePolicy=fail,groups="",resources=deployments,verbs=create;update,versions=v1,name=controller-webhook.nicholasjackson.io,sideEffects=None,admissionReviewVersions=v1

// deploymentAdmission controls wether a new deployment is accepted by Kubernetes.
// Deployments should not be permitted when there is an active release.
type deploymentAdmission struct {
	Client    client.Client
	decoder   *admission.Decoder
	admission admissionController.Admission
}

func NewDeploymentAdmission(client client.Client, admission admissionController.Admission) *deploymentAdmission {
	return &deploymentAdmission{Client: client, admission: admission}
}

func (a *deploymentAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {
	deployment := &appsv1.Deployment{}

	err := a.decoder.Decode(req, deployment)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// check if the deployment is allowed
	resp, err := a.admission.Check(
		ctx,
		deployment.Name,
		deployment.Namespace,
		deployment.Labels,
		deployment.ResourceVersion,
		interfaces.RuntimePlatformKubernetes)

	if err != nil && resp == admissionController.AdmissionError {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if resp == admissionController.AdmissionGranted {
		return admission.Allowed("")
	}

	return admission.Denied(err.Error())
}

// podAnnotator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *deploymentAdmission) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
