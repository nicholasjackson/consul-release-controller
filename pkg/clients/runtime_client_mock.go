package clients

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type RuntimeClientMock struct {
	mock.Mock
}

func (rc *RuntimeClientMock) GetDeployment(ctx context.Context, name, namespace string) (*Deployment, error) {
	args := rc.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (rc *RuntimeClientMock) GetDeploymentWithSelector(ctx context.Context, selector, namespace string) (*Deployment, error) {
	args := rc.Called(ctx, selector, namespace)

	if d, ok := args.Get(0).(*Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (rc *RuntimeClientMock) UpdateDeployment(ctx context.Context, deployment *Deployment) error {
	args := rc.Called(ctx, deployment)

	return args.Error(0)
}

func (rc *RuntimeClientMock) CloneDeployment(ctx context.Context, existingDeployment *Deployment, newDeployment *Deployment) error {
	args := rc.Called(ctx, existingDeployment, newDeployment)

	return args.Error(0)
}

func (rc *RuntimeClientMock) DeleteDeployment(ctx context.Context, name, namespace string) error {
	args := rc.Called(ctx, name, namespace)

	return args.Error(0)
}

func (rc *RuntimeClientMock) GetHealthyDeployment(ctx context.Context, name, namespace string) (*Deployment, error) {
	args := rc.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}
