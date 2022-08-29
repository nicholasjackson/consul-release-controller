package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	admissionController "github.com/nicholasjackson/consul-release-controller/pkg/controllers"
	admissionMock "github.com/nicholasjackson/consul-release-controller/pkg/controllers/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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

func setupAdmission(t *testing.T) (*deploymentAdmission, *admissionMock.Admission) {
	am := &admissionMock.Admission{}
	am.On(
		"Check",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(admissionController.AdmissionGranted, nil)

	da := NewDeploymentAdmission(nil, am)

	decoder, err := admission.NewDecoder(scheme)
	require.NoError(t, err)

	da.InjectDecoder(decoder)

	return da, am
}

func TestDecodeBadDeploymentReturnsErrorOnBadPayload(t *testing.T) {
	req := createAdmissionRequest(false)
	da, _ := setupAdmission(t)

	req.Object.Raw = nil

	ar := da.Handle(context.Background(), req)
	require.False(t, ar.Allowed)
	require.Equal(t, int32(http.StatusBadRequest), ar.Result.Code)
}

func TestDecodeBadDeploymentReturnsErrorOnCheckError(t *testing.T) {
	req := createAdmissionRequest(false)
	da, cm := setupAdmission(t)

	testutils.ClearMockCall(&cm.Mock, "Check")
	cm.On("Check", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		admissionController.AdmissionError,
		fmt.Errorf("boom"),
	)

	ar := da.Handle(context.Background(), req)
	require.False(t, ar.Allowed)
	require.Equal(t, int32(http.StatusInternalServerError), ar.Result.Code)
}

func TestDecodeBadDeploymentReturnsAllowedOnSuccesfulCheck(t *testing.T) {
	req := createAdmissionRequest(false)
	da, _ := setupAdmission(t)

	ar := da.Handle(context.Background(), req)
	require.True(t, ar.Allowed)
}

func TestDecodeBadDeploymentReturnsDeniedOnFailedCheck(t *testing.T) {
	req := createAdmissionRequest(false)
	da, cm := setupAdmission(t)

	testutils.ClearMockCall(&cm.Mock, "Check")
	cm.On(
		"Check",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(admissionController.AdmissionGranted, nil)

	ar := da.Handle(context.Background(), req)
	require.True(t, ar.Allowed)
}
