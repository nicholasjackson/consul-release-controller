package kubernetes

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupPlugin(t *testing.T) (*Plugin, *clients.KubernetesMock) {
	l := hclog.Default()

	one := int32(1)

	dep := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "testnamespace",
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
		Spec:   appsv1.DeploymentSpec{Replicas: &one},
	}

	km := &clients.KubernetesMock{}
	km.On("GetDeployment", mock.Anything, mock.Anything).Return(dep, nil)
	km.On("UpsertDeployment", mock.Anything).Return(nil)

	p := &Plugin{}

	p.log = l
	p.kubeClient = km
	p.config = &PluginConfig{Deployment: "test", Namespace: "testnamespace"}

	return p, km
}

func TestDeployCreatesCloneWhenOriginalFound(t *testing.T) {
	p, km := setupPlugin(t)

	p.Deploy(context.Background())

	km.AssertCalled(t, "GetDeployment", "test", "testnamespace")
	km.AssertCalled(t, "UpsertDeployment", mock.Anything)

	dep := km.Calls[1].Arguments[0].(*appsv1.Deployment)
	require.Equal(t, "test-primary", dep.Name)
	require.Equal(t, "testnamespace", dep.Namespace)
}
