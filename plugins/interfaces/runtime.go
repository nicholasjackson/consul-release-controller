package interfaces

import (
	"context"
)

type RuntimeDeploymentStatus string

const (
	RuntimeDeploymentUpdate        RuntimeDeploymentStatus = "runtime_deployment_update"
	RuntimeDeploymentNoAction      RuntimeDeploymentStatus = "runtime_deployment_no_action"
	RuntimeDeploymentNotFound      RuntimeDeploymentStatus = "runtime_deployment_not_found"
	RuntimeDeploymentInternalError RuntimeDeploymentStatus = "runtime_deployment_internal_error"
	RuntimeDeploymentVersionLabel                          = "consul-release-controller-version"
)

// RuntimeBaseConfig is the base configuration that all runtime plugins must implement
type RuntimeBaseConfig struct {
	// DeploymentSelector is used to determine which deployments can trigger a release
	// can contain regular expressions
	DeploymentSelector string `hcl:"deployment" json:"deployment"`
	// Namespace for the deployment that triggers a release
	Namespace string `hcl:"namespace" json:"namespace"`
}

// RuntimeBaseState is the basic state that all runtime plugins need to implement
type RuntimeBaseState struct {
	// CandidateName is the full name of the active candidate deployment
	CandidateName string `hcl:"candidate_name" json:"candidate_name"`
	// PrimaryName is the full name of the active primary deployment
	PrimaryName string `hcl:"primary_name" json:"primary_name"`
}

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	Configurable

	// BaseConfig returns the base Runtime config
	// all Runtime plugins should embed RuntimeBaseConfig in their own config
	BaseConfig() RuntimeBaseConfig

	// BaseState returns the base Runtime state
	// all Runtime plugins should embed RuntimeBaseState in their own state
	BaseState() RuntimeBaseState

	// Create a copy of the active deployment whos lifecycle will be owned
	// by the release controller
	InitPrimary(ctx context.Context, releaseName string) (RuntimeDeploymentStatus, error)

	// Promote the new test version to primary
	// returns RunTimeDeploymentUpdate when the canary has been successfully promoted to primary
	// returns RuntimeDeploymentNotFound when the canary does not exist
	// returned errors indicate an internal error
	PromoteCandidate(ctx context.Context) (RuntimeDeploymentStatus, error)

	// Cleanup the test version saving resources
	// Candidate
	RemoveCandidate(ctx context.Context) error

	// RestoreOriginal re-instates the original deployment
	RestoreOriginal(ctx context.Context) error

	// RemovePrimary removes the Primary deployment that is a clone of the original
	RemovePrimary(ctx context.Context) error
}
