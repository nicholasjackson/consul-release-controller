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
	Deployment string `hcl:"deployment" json:"deployment"`
	Namespace  string `hcl:"namespace" json:"namespace"`
}

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	Configurable

	// BaseConfig returns the base Runtime config
	// all Runtime plugins should embed RuntimeBaseConfig in their own config
	BaseConfig() RuntimeBaseConfig

	// Deploy the new test version to the platform
	InitPrimary(ctx context.Context) (RuntimeDeploymentStatus, error)

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
