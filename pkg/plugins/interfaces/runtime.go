package interfaces

import (
	"context"
	"fmt"
)

const (
	RuntimePlatformKubernetes = "kubernetes"
	RuntimePlatformNomad      = "nomad"
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

	// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
	CandidateSubsetFilter() string

	// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
	PrimarySubsetFilter() string
}

const (
	DeploymentNotFound   = "deployment_not_found"
	DeploymentNotHealthy = "deployment_not_healthy"
)

var ErrDeploymentNotFound = fmt.Errorf(DeploymentNotFound)
var ErrDeploymentNotHealthy = fmt.Errorf(DeploymentNotHealthy)

// Deployment is a type that defines an abstract deployment
type Deployment struct {
	// Name of the deployment
	Name string
	// Namespace for the deployment
	Namespace string
	// ResourceVersion of the deployment
	ResourceVersion string
	// Meta data associated with the deployment, e.g. translates to labels in Kubernetes or Meta in Nomad
	Meta map[string]string
	// Instances of a deployment to run
	Instances int
}

// NewDeployment creates a new deployment
func NewDeployment() *Deployment {
	return &Deployment{
		Meta: map[string]string{},
	}
}

// RuntimeClient is a high level functional interface for interacting with the runtime APIs like Kubernetes
type RuntimeClient interface {
	// GetDeployment returns a Kubernetes deployment matching the given name and
	// namespace.
	// If the deployment does not exist a DeploymentNotFound error will be returned
	// and a nil deployments
	// Any other error than DeploymentNotFound can be treated like an internal error
	// in executing the request
	GetDeployment(ctx context.Context, name, namespace string) (*Deployment, error)

	// GetDeploymentWithSelector returns the first deployment whos name and namespace match the given
	// regular expression and namespace.
	GetDeploymentWithSelector(ctx context.Context, selector, namespace string) (*Deployment, error)

	// UpdateDeployment updates an active deployment, settng metadata or scale parameters, name and namespace can not be changed
	// returns DeploymentNotFound if the deployment does not exist
	UpdateDeployment(ctx context.Context, deployment *Deployment) error

	// CloneDeployment creates a clone of the existing deployment using the details provided in new deployment
	CloneDeployment(ctx context.Context, existingDeployment *Deployment, newDeployment *Deployment) error

	// DeleteDeployment deletes the given Kubernetes Deployment
	DeleteDeployment(ctx context.Context, name, namespace string) error

	// GetHealthyDeployment blocks until a healthy deployment is found or the process times out
	// returns the Deployment an a nil error on success
	// returns a nil deployment and a ErrDeploymentNotFound error when the deployment does not exist
	// returns a nill deployment and a ErrDeploymentNotHealthy error when the deployment exists but is not in a healthy state
	// any other error type signifies an internal error
	GetHealthyDeployment(ctx context.Context, name, namespace string) (*Deployment, error)

	// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
	CandidateSubsetFilter() string

	// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
	PrimarySubsetFilter() string
}
