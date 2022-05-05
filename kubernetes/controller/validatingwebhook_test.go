package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupAdmission(t *testing.T, deploymentName, namespace string) (*deploymentAdmission, *mocks.Mocks) {
	pm, mm := mocks.BuildMocks(t)

	pc := &kubernetes.PluginConfig{}
	pc.DeploymentSelector = deploymentName
	pc.Namespace = namespace

	pcd, _ := json.Marshal(pc)

	testutils.ClearMockCall(&mm.StoreMock.Mock, "ListReleases")

	mm.StoreMock.On("ListReleases", &interfaces.ListOptions{"kubernetes"}).Return(
		[]*models.Release{
			&models.Release{
				Name: "test",
				Runtime: &models.PluginConfig{
					Name:   "kubernetes",
					Config: pcd,
				},
			},
		},
		nil,
	)

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateIdle)

	da := NewDeploymentAdmission(nil, pm, pm.GetLogger())

	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)

	da.InjectDecoder(decoder)

	return da, mm
}

func createAdmissionRequest(withVersionLabels bool) admission.Request {
	ar := admission.Request{}
	ar.AdmissionRequest.Name = "test-deployment"

	dep := &appsv1.Deployment{}
	dep.Namespace = "default"
	dep.Name = "test-deployment"
	dep.Labels = map[string]string{"app": "test"}

	if withVersionLabels {
		dep.Labels["consul-release-controller-version"] = "1"
		dep.ResourceVersion = "1"
	}

	data, _ := json.Marshal(dep)

	ar.Object.Raw = data

	return ar
}

func TestIgnoresDeploymentModifiedByControllerWhenActive(t *testing.T) {
	ar := createAdmissionRequest(true)
	d, mm := setupAdmission(t, "test-deployment", "default")

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestDoesNothingForNewDeploymentWithNamespaceMismatch(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "mine")

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenIdle(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "default")

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
	mm.StoreMock.AssertCalled(t, "UpsertRelease", mock.Anything)

	// check that the kubernetes deployment name is saved to the release config
	rbc := getUpsertReleaseConfig(mm.StoreMock.Mock)
	require.NotNil(t, rbc)
	require.Equal(t, "test-deployment", rbc.CandidateName)
}

func TestReturnsErrorWhenNewDeploymentUpsertReleaseFails(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StoreMock.Mock, "UpsertRelease")
	mm.StoreMock.On("UpsertRelease", mock.Anything).Return(fmt.Errorf("boom"))

	resp := d.Handle(context.TODO(), ar)
	require.False(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
	mm.StoreMock.AssertCalled(t, "UpsertRelease", mock.Anything)
}

func TestAddsRegExpWordBoundaryAndFailsMatchWhenNotPresent(t *testing.T) {
	ar := createAdmissionRequest(false)

	// a regexp without a word boundary would match, check we add
	// the word boundary when not present
	d, mm := setupAdmission(t, "test-", "default")

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenIdleAndUsingRegularExpressions(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-(.*)", "default")

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenFailed(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateFail)

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestReturnsAllowedWhenReleaseNotFound(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StoreMock.Mock, "ListReleases")
	mm.StoreMock.On("ListReleases", &interfaces.ListOptions{"kubernetes"}).Return(
		[]*models.Release{},
		nil,
	)

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestReturnsDeniedWhenReleaseActive(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateMonitor)

	resp := d.Handle(context.TODO(), ar)
	require.False(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func getUpsertReleaseConfig(mock mock.Mock) *interfaces.RuntimeBaseConfig {
	for _, c := range mock.Calls {
		if c.Method == "UpsertRelease" {
			if dep, ok := c.Arguments.Get(0).(*models.Release); ok {
				rbc := &interfaces.RuntimeBaseConfig{}
				json.Unmarshal(dep.Runtime.Config, rbc)
				return rbc
			}
		}
	}

	return nil
}
