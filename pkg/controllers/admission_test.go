package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/runtime"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAdmission(t *testing.T, deploymentName, namespace string) (Admission, *mocks.Mocks) {
	pm, mm := mocks.BuildMocks(t)

	pc := &runtime.PluginConfig{}
	pc.DeploymentSelector = deploymentName
	pc.Namespace = namespace

	pcd, _ := json.Marshal(pc)

	testutils.ClearMockCall(&mm.StoreMock.Mock, "ListReleases")

	mm.StoreMock.On("ListReleases", &interfaces.ListOptions{Runtime: "kubernetes"}).Return(
		[]*models.Release{
			&models.Release{
				Name: deploymentName,
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

	da := NewAdmission(pm, pm.GetLogger())

	return da, mm
}

func TestIgnoresDeploymentModifiedByControllerWhenActive(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{interfaces.RuntimeDeploymentVersionLabel: "2"}, "2", "kubernetes")
	require.NoError(t, err)
	require.Equal(t, resp, AdmissionGranted)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestDoesNothingForNewDeploymentWithNamespaceMismatch(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "mine")

	resp, err := d.Check(context.TODO(), "test-deployment", "other", map[string]string{interfaces.RuntimeDeploymentVersionLabel: "2"}, "2", "kubernetes")
	require.NoError(t, err)
	require.Equal(t, resp, AdmissionGranted)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenIdle(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
	mm.StoreMock.AssertCalled(t, "UpsertState", mock.Anything)

	// check that the kubernetes deployment name is saved to the release config
	rbc := getUpsertReleaseState(&mm.StoreMock.Mock)
	require.NotNil(t, rbc)
	require.Equal(t, "test-deployment", rbc.CandidateName)
}

func TestReturnsErrorWhenNewDeploymentUpsertReleaseFails(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StoreMock.Mock, "UpsertState")
	mm.StoreMock.On("UpsertState", mock.Anything).Return(fmt.Errorf("boom"))

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionError)
	require.Error(t, err)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
	mm.StoreMock.AssertCalled(t, "UpsertState", mock.Anything)
}

func TestAddsRegExpWordBoundaryAndFailsMatchWhenNotPresent(t *testing.T) {
	// a regexp without a word boundary would match, check we add
	// the word boundary when not present
	d, mm := setupAdmission(t, "test-", "default")

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenIdleAndUsingRegularExpressions(t *testing.T) {
	d, mm := setupAdmission(t, "test-(.*)", "default")

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenFailed(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateFail)

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestReturnsAllowedWhenReleaseNotFound(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StoreMock.Mock, "ListReleases")
	mm.StoreMock.On("ListReleases", &interfaces.ListOptions{Runtime: "kubernetes"}).Return(
		[]*models.Release{},
		nil,
	)

	resp, err := d.Check(context.TODO(), "test-deployment", "default", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestReturnsDeniedWhenReleaseActive(t *testing.T) {
	d, mm := setupAdmission(t, "test-deployment", "default")

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateMonitor)

	resp, err := d.Check(context.TODO(), "test", "other", map[string]string{}, "2", "kubernetes")
	require.Equal(t, resp, AdmissionGranted)
	require.NoError(t, err)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func getUpsertReleaseConfig(mock *mock.Mock) *interfaces.RuntimeBaseConfig {
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

func getUpsertReleaseState(mock *mock.Mock) *interfaces.RuntimeBaseState {
	for _, c := range mock.Calls {
		if c.Method == "UpsertState" {
			if dep, ok := c.Arguments.Get(0).([]byte); ok {
				rbc := &interfaces.RuntimeBaseState{}
				json.Unmarshal(dep, rbc)
				return rbc
			}
		}
	}

	return nil
}
