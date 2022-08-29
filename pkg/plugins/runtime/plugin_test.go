package runtime

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var mockDep = interfaces.Deployment{
	Name:      "test-deployment",
	Namespace: "testnamespace",
	Instances: 1,
}

var mockCloneDep = interfaces.Deployment{
	Name:      "test-deployment-primary",
	Namespace: "testnamespace",
	Instances: 1,
}

func setupPlugin(t *testing.T) (*Plugin, *clients.RuntimeClientMock, interfaces.Deployment, interfaces.Deployment) {
	l := hclog.New(&hclog.LoggerOptions{Level: hclog.Debug})

	km := &clients.RuntimeClientMock{}

	sm := &mocks.StoreMock{}
	sm.On("GetState").Return([]byte(testState), nil)
	sm.On("UpsertState", mock.Anything).Return(nil)

	p := &Plugin{}

	p.log = l
	p.client = km
	p.store = sm

	p.config = &PluginConfig{}
	p.config.DeploymentSelector = "test-(.*)"

	p.state = &PluginState{}
	p.state.CandidateName = "test-deployment"
	p.state.PrimaryName = "test-deployment-primary"
	p.config.Namespace = "testnamespace"

	return p, km, mockDep, mockCloneDep
}

func TestConfigureLoadsConfig(t *testing.T) {
	km := &clients.RuntimeClientMock{}
	p, _ := New(km)
	sm := &mocks.StoreMock{}
	sm.On("GetState").Return([]byte(testState), nil)

	os.Setenv("KUBECONFIG", testutils.GetTestFilePath(t, "kubeconfig.yaml"))

	err := p.Configure([]byte(testConfig), hclog.NewNullLogger(), sm)
	require.NoError(t, err)

	require.Equal(t, "testnamespace", p.config.Namespace)
	require.Equal(t, "api-deployment", p.config.DeploymentSelector)

	require.Equal(t, "api-primary", p.state.PrimaryName)
	require.Equal(t, "api-deployment-v1", p.state.CandidateName)
}

func TestInitPrimaryDoesNothingWhenPrimaryExists(t *testing.T) {
	p, km, dep, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(&cloneDep, nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Return(&dep, nil)

	status, err := p.InitPrimary(context.Background(), "test-deployment")
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
	km.On("GetDeploymentWithSelector", mock.Anything, "test-(.*)", "testnamespace").Return(nil, fmt.Errorf("Candidate not Found"))

	status, err := p.InitPrimary(context.Background(), "test-deployment")
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentNoAction, status)
}

func TestInitPrimaryCreatesPrimaryWhenCandidateExists(t *testing.T) {
	p, km, dep, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(nil, fmt.Errorf("Primary not found"))
	km.On("GetDeploymentWithSelector", mock.Anything, "test-(.*)", "testnamespace").Return(&dep, nil)
	km.On("CloneDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(&cloneDep, nil)

	status, err := p.InitPrimary(context.Background(), "test-deployment")
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentUpdate, status)

	km.AssertCalled(t, "CloneDeployment", mock.Anything, mock.Anything, mock.Anything)

	// check that the runtimedeploymentversion label is added to ensure the validating webhook ignores this deployment
	depArg := getCloneDeployment(&km.Mock)
	require.Equal(t, "1", depArg.Meta[interfaces.RuntimeDeploymentVersionLabel])
}

func TestPromoteCandidateDoesNothingWhenCandidateNotExists(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Return(nil, interfaces.ErrDeploymentNotFound)

	status, err := p.PromoteCandidate(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentNotFound, status)

	km.AssertCalled(t, "GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace")
}

func TestPromoteCandidateDeletesExistingPrimaryAndUpserts(t *testing.T) {
	p, km, dep, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(&dep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Return(nil)
	km.On("CloneDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(&dep, nil)

	status, err := p.PromoteCandidate(context.Background())
	require.NoError(t, err)
	require.Equal(t, interfaces.RuntimeDeploymentUpdate, status)

	km.AssertCalled(t, "DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace")
	km.AssertCalled(t, "CloneDeployment", mock.Anything, mock.Anything, mock.Anything)

	// check that the runtimedeploymentversion label is added to ensure the validating webhook ignores this deployment
	depArg := getCloneDeployment(&km.Mock)
	require.Equal(t, "1", depArg.Meta[interfaces.RuntimeDeploymentVersionLabel])
}

func TestRemoveCandidateDoesNothingWhenCandidateNotFound(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil, interfaces.ErrDeploymentNotFound)

	err := p.RemoveCandidate(context.Background())
	require.NoError(t, err)

	km.AssertNotCalled(t, "CloneDeployment", mock.Anything, mock.Anything)
}

func TestRemoveCandidateScalesWhenCandidateFound(t *testing.T) {
	p, km, dep, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(&dep, nil)
	km.On("UpdateDeployment", mock.Anything, mock.Anything).Once().Return(nil)

	err := p.RemoveCandidate(context.Background())
	require.NoError(t, err)

	require.Equal(t, 0, dep.Instances)
	km.AssertCalled(t, "UpdateDeployment", mock.Anything, &dep)
}

func TestRestoreDoesNothingWhenNoPrimaryFound(t *testing.T) {
	p, km, _, _ := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(nil, interfaces.ErrDeploymentNotFound)

	err := p.RestoreOriginal(context.Background())
	require.NoError(t, err)
}

func TestRestoreCallsDeleteWhenPrimaryFound(t *testing.T) {
	p, km, _, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(&cloneDep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil)
	km.On("CloneDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	km.On("GetHealthyDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(nil, nil)

	err := p.RestoreOriginal(context.Background())
	require.NoError(t, err)

	// check that the labels are removed
	depArg := getCloneDeployment(&km.Mock)
	require.Equal(t, depArg.Meta[interfaces.RuntimeDeploymentVersionLabel], "")
}

func TestRestoreProceedesWhenExistingCandidateNotFound(t *testing.T) {
	p, km, _, cloneDep := setupPlugin(t)

	testutils.ClearMockCall(&km.Mock, "GetDeployment")
	km.On("GetDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(&cloneDep, nil)
	km.On("DeleteDeployment", mock.Anything, "test-deployment", "testnamespace").Once().Return(interfaces.ErrDeploymentNotFound)
	km.On("CloneDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
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
	km.On("DeleteDeployment", mock.Anything, "test-deployment-primary", "testnamespace").Once().Return(interfaces.ErrDeploymentNotFound)

	err := p.RemovePrimary(context.Background())
	require.NoError(t, err)
}

func getCloneDeployment(mock *mock.Mock) *interfaces.Deployment {
	for _, c := range mock.Calls {
		if c.Method == "CloneDeployment" {
			if dep, ok := c.Arguments.Get(2).(*interfaces.Deployment); ok {
				return dep
			}
		}
	}

	return nil
}

var testConfig = `
{
  "deployment": "api-deployment",
  "namespace":"testnamespace"
}
`

var testState = `
{
  "candidate_name": "api-deployment-v1",
  "primary_name":"api-primary"
}
`
