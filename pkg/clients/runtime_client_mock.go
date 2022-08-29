package clients

import (
	"context"

	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type RuntimeClientMock struct {
	mock.Mock
}

func (rc *RuntimeClientMock) GetDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	args := rc.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*interfaces.Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (rc *RuntimeClientMock) GetDeploymentWithSelector(ctx context.Context, selector, namespace string) (*interfaces.Deployment, error) {
	args := rc.Called(ctx, selector, namespace)

	if d, ok := args.Get(0).(*interfaces.Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (rc *RuntimeClientMock) UpdateDeployment(ctx context.Context, deployment *interfaces.Deployment) error {
	args := rc.Called(ctx, deployment)

	return args.Error(0)
}

func (rc *RuntimeClientMock) CloneDeployment(ctx context.Context, existingDeployment *interfaces.Deployment, newDeployment *interfaces.Deployment) error {
	args := rc.Called(ctx, existingDeployment, newDeployment)

	return args.Error(0)
}

func (rc *RuntimeClientMock) DeleteDeployment(ctx context.Context, name, namespace string) error {
	args := rc.Called(ctx, name, namespace)

	return args.Error(0)
}

func (rc *RuntimeClientMock) GetHealthyDeployment(ctx context.Context, name, namespace string) (*interfaces.Deployment, error) {
	args := rc.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*interfaces.Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify candidate instances
func (rc *RuntimeClientMock) CandidateSubsetFilter() string {
	rc.Called()

	return "primary"
}

// Returns the Consul resolver subset filter that should be used for this runtime to identify the primary instances
func (rc *RuntimeClientMock) PrimarySubsetFilter() string {
	rc.Called()

	return "candidate"
}
