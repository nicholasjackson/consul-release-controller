package controller

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func setupAdmission(t *testing.T) (*deploymentAdmission, *mocks.Mocks) {
	pm, mm := mocks.BuildMocks(t)

	pc := &kubernetes.PluginConfig{}
	pc.Deployment = "test-deployment"

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
	d, mm := setupAdmission(t)

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenIdle(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t)

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestCallsDeployForNewDeploymentWhenFailed(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t)

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateFail)

	resp := d.Handle(context.TODO(), ar)
	require.True(t, resp.Allowed)
	mm.StateMachineMock.AssertCalled(t, "Deploy")
}

func TestReturnsAllowedWhenReleaseNotFound(t *testing.T) {
	ar := createAdmissionRequest(false)
	d, mm := setupAdmission(t)

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
	d, mm := setupAdmission(t)

	testutils.ClearMockCall(&mm.StateMachineMock.Mock, "CurrentState")
	mm.StateMachineMock.On("CurrentState").Return(interfaces.StateMonitor)

	resp := d.Handle(context.TODO(), ar)
	require.False(t, resp.Allowed)
	mm.StateMachineMock.AssertNotCalled(t, "Deploy")
}
