package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/clients"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var one = int32(1)
var mockDep = &appsv1.Deployment{
	ObjectMeta: v1.ObjectMeta{
		Name:      "test-deployment",
		Namespace: "testnamespace",
	},
	Status: appsv1.DeploymentStatus{ReadyReplicas: 1, UnavailableReplicas: 0, AvailableReplicas: 1},
	Spec:   appsv1.DeploymentSpec{Replicas: &one},
}

var mockCloneDep = &appsv1.Deployment{
	ObjectMeta: v1.ObjectMeta{
		Name:      "test-deployment-primary",
		Namespace: "testnamespace",
	},
	Status: appsv1.DeploymentStatus{ReadyReplicas: 1, UnavailableReplicas: 0, AvailableReplicas: 1},
	Spec:   appsv1.DeploymentSpec{Replicas: &one},
}

func setupPlugin(t *testing.T) (*Plugin, *clients.KubernetesMock, *appsv1.Deployment, *appsv1.Deployment) {

	retryTimeout = 10 * time.Millisecond
	retryInterval = 1 * time.Millisecond

	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug})

	km := &clients.KubernetesMock{}

	p := &Plugin{}

	p.log = l
	p.kubeClient = km
	p.config = &PluginConfig{}
	p.config.Deployment = "test-deployment"
	p.config.Namespace = "testnamespace"

	return p, km, mockDep.DeepCopy(), mockCloneDep.DeepCopy()
}

func TestInitPrimaryDoesNothingWhenPrimaryExists(t *testing.T) {
	p, km, dep, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(cloneDep, nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Return(dep, nil)

	status, err := p.InitPrimary(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentNoAction, status)

	km.AssertCalled(t, "GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace")
	km.AssertNotCalled(t, "GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace")
	km.AssertNotCalled(t, "UpsertDeployment", mock.Anything, mock.Anything)
}

func TestInitPrimaryDoesNothingWhenCandidateDoesNotExist(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(nil, fmt.Errorf("Primary not found"))
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Return(nil, fmt.Errorf("Candidate not Found"))

	status, err := p.InitPrimary(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentNoAction, status)
}

func TestInitPrimaryCreatesPrimaryWhenCandidateExists(t *testing.T) {
	p, km, dep, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(nil, fmt.Errorf("Primary not found"))
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Return(dep, nil)
	km.On("UpsertDeployment", mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(cloneDep, nil)

	status, err := p.InitPrimary(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentUpdate, status)

	km.AssertCalled(t, "UpsertDeployment", mock.Anything, mock.Anything)

	// check that the runtimedeploymentversion label is added to ensure the validating webhook ignores this deployment
	depArg := getUpsertDeployment(km.Mock)
	require.Equal(t, depArg.Labels[interfaces.RuntimeDeploymentVersionLabel], "1")
}

func TestPromoteCandidateDoesNothingWhenCandidateNotExists(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Return(nil, clients.ErrDeploymentNotFound)

	status, err := p.PromoteCandidate(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentNotFound, status)

	km.AssertCalled(t, "GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace")
}

func TestPromoteCandidateDeletesExistingPrimaryAndUpserts(t *testing.T) {
	p, km, dep, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(dep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(nil)
	km.On("UpsertDeployment", mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(dep, nil)

	status, err := p.PromoteCandidate(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentUpdate, status)

	km.AssertCalled(t, "DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace")
	km.AssertCalled(t, "UpsertDeployment", mock.Anything, mock.Anything)

	// check that the runtimedeploymentversion label is added to ensure the validating webhook ignores this deployment
	depArg := getUpsertDeployment(km.Mock)
	require.Equal(t, depArg.Labels[interfaces.RuntimeDeploymentVersionLabel], "1")
}

func TestRemoveCandidateDoesNothingWhenCandidateNotFound(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil, clients.ErrDeploymentNotFound)

	err := p.RemoveCandidate(context.Background())
	require.NoError(t, err)

	km.AssertNotCalled(t, "UpsertDeployment", mock.Anything, mock.Anything)
}

func TestRemoveCandidateScalesWhenCandidateFound(t *testing.T) {
	p, km, dep, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(dep, nil)
	km.On("UpsertDeployment", mock.Anything, mock.Anything).Once().Return(nil)

	err := p.RemoveCandidate(context.Background())
	require.NoError(t, err)

	require.Equal(t, int32(0), *dep.Spec.Replicas)
	km.AssertCalled(t, "UpsertDeployment", mock.Anything, dep)
}

func TestRestoreDoesNothingWhenNoPrimaryFound(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(nil, clients.ErrDeploymentNotFound)

	err := p.RestoreOriginal(context.Background())
	require.NoError(t, err)
}

func TestRestoreCallsDeleteWhenPrimaryFound(t *testing.T) {
	p, km, _, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(cloneDep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil)
	km.On("UpsertDeployment", mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil, nil)

	err := p.RestoreOriginal(context.Background())
	require.NoError(t, err)

	// check that the labels are removed
	depArg := getUpsertDeployment(km.Mock)
	require.Equal(t, depArg.Labels[interfaces.RuntimeDeploymentVersionLabel], "")
}

func TestRestoreProceedesWhenExistingCandidateNotFound(t *testing.T) {
	p, km, _, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(cloneDep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(clients.ErrDeploymentNotFound)
	km.On("UpsertDeployment", mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil, nil)

	err := p.RestoreOriginal(context.Background())
	require.NoError(t, err)
}

func TestRemovePrimaryCallsDelete(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(nil)

	err := p.RemovePrimary(context.Background())
	require.NoError(t, err)
}

func TestRemovePrimaryReturnsErrorWhenDeleteIsGenericError(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(fmt.Errorf("test"))

	err := p.RemovePrimary(context.Background())
	require.Error(t, err)
}

func TestRemovePrimaryReturnsNoErrorWhenDeleteIsNotFoundError(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(clients.ErrDeploymentNotFound)

	err := p.RemovePrimary(context.Background())
	require.NoError(t, err)
}

func getUpsertDeployment(mock mock.Mock) *appsv1.Deployment {
	for _, c := range mock.Calls {
		if c.Method == "UpsertDeployment" {
			if dep, ok := c.Arguments.Get(1).(*appsv1.Deployment); ok {
				return dep
			}
		}
	}

	return nil
}
