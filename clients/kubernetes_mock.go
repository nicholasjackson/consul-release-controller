package clients

import (
	"context"

	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
)

// KubernetesMock is a mock implementation of the Kubernetes interface, this can
// be used when writing tests
type KubernetesMock struct {
	mock.Mock
}

func (k *KubernetesMock) GetDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	args := k.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*appsv1.Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}

func (k *KubernetesMock) UpsertDeployment(ctx context.Context, d *appsv1.Deployment) error {
	args := k.Called(ctx, d)

	return args.Error(0)
}

func (k *KubernetesMock) DeleteDeployment(ctx context.Context, name, namespace string) error {
	args := k.Called(ctx, name, namespace)

	return args.Error(0)
}

func (k *KubernetesMock) GetHealthyDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	args := k.Called(ctx, name, namespace)

	if d, ok := args.Get(0).(*appsv1.Deployment); ok {
		return d, args.Error(1)
	}

	return nil, args.Error(1)
}
