package interfaces

import (
	"context"
	"encoding/json"
)

type RuntimeDeploymentStatus string

const (
	RuntimeDeploymentUpdate   RuntimeDeploymentStatus = "runtime_deployment_update"
	RuntimeDeploymentNotFound RuntimeDeploymentStatus = "runtime_deployment_not_found"
)

// RuntimeBaseConfig is the base configuration that all runtime plugins must implement
type RuntimeBaseConfig struct {
	Deployment string `hcl:"deployment" json:"deployment"`
	Namespace  string `hcl:"namespace" json:"namespace"`
}

// Runtime defines an interface that all concrete platforms like Kubernetes must
// implement
type Runtime interface {
	// Configure the plugin with the given json
	Configure(config json.RawMessage) error

	// BaseConfig returns the base Runtime config
	// all Runtime plugins should embed RuntimeBaseConfig in their own config
	BaseConfig() RuntimeBaseConfig

	// Deploy the new test version to the platform
	Deploy(ctx context.Context, status RuntimeDeploymentStatus) error

	// Promote the new test version to primary
	Promote(ctx context.Context) error

	// Cleanup the test version saving resources
	Cleanup(ctx context.Context) error

	// Destroy removes any configuration that was created with the Deploy method
	Destroy(ctx context.Context) error
}
