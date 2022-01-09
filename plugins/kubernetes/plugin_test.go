package kubernetes

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var one = int32(1)
var dep = &appsv1.Deployment{
	ObjectMeta: v1.ObjectMeta{
		Name:      "test",
		Namespace: "testnamespace",
	},
	Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
	Spec:   appsv1.DeploymentSpec{Replicas: &one},
}

var cloneDep = &appsv1.Deployment{
	ObjectMeta: v1.ObjectMeta{
		Name:      "test-primary",
		Namespace: "testnamespace",
	},
	Status: appsv1.DeploymentStatus{ReadyReplicas: 1},
	Spec:   appsv1.DeploymentSpec{Replicas: &one},
}

func setupPlugin(t *testing.T) (*Plugin, *clients.KubernetesMock) {
	l := hclog.Default()

	km := &clients.KubernetesMock{}
	km.On("GetDeployment", "test", mock.Anything).Return(dep, nil)
	km.On("GetDeployment", "test-primary", mock.Anything).Once().Return(nil, nil)
	km.On("GetDeployment", "test-primary", mock.Anything).Once().Return(cloneDep, nil)
	km.On("UpsertDeployment", mock.Anything).Return(nil)
	km.On("DeleteDeployment", mock.Anything, mock.Anything).Return(nil)

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

	dep := km.Calls[2].Arguments[0].(*appsv1.Deployment)
	require.Equal(t, "test-primary", dep.Name)
	require.Equal(t, "testnamespace", dep.Namespace)
}

func TestDeployDoesNotCreatesCloneWhenPrimaryExists(t *testing.T) {
	p, km := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", "test", mock.Anything).Return(dep, nil)
	km.On("GetDeployment", "test-primary", mock.Anything).Once().Return(cloneDep, nil)
	km.On("GetDeployment", "test-primary", mock.Anything).Once().Return(cloneDep, nil)

	p.Deploy(context.Background())

	km.AssertCalled(t, "GetDeployment", "test", "testnamespace")
	km.AssertNotCalled(t, "UpsertDeployment", mock.Anything)
}

func TestPromoteClonesCanaryToCreateNewPrimary(t *testing.T) {
	p, km := setupPlugin(t)

	err := p.Promote(context.Background())
	require.NoError(t, err)

	km.AssertCalled(t, "DeleteDeployment", mock.Anything, mock.Anything)
	km.AssertCalled(t, "UpsertDeployment", mock.Anything)
}
