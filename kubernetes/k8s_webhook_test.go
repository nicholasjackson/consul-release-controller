package kubernetes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupWebhook(t *testing.T) (func(w http.ResponseWriter, r *http.Request), *state.MockStore, *mocks.ProviderMock, *mocks.Mocks) {
	pp, pm := mocks.BuildMocks(t)
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}

	wh, _ := NewK8sWebhook(l, m, s, pp)

	return wh.Mutating(), s, pp, pm
}

func setupReleases(t *testing.T, pp *mocks.ProviderMock, pm *mocks.Mocks, s *state.MockStore, name string) *models.Release {
	depData := testutils.GetTestData(t, "valid_kubernetes_release.json")

	// modify the step delay for tests
	models.StepDelay = 1 * time.Millisecond

	// the release is create when configuration is save by the server
	// by the time the kubernetes hook runs, this object will exist
	dep := &models.Release{}
	dep.FromJsonBody(ioutil.NopCloser(bytes.NewBuffer(depData)))
	dep.CurrentState = models.StateIdle // set the initial state to idle
	dep.Build(pp)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "BaseConfig")

	pc := interfaces.RuntimeBaseConfig{}
	pc.Deployment = name
	pc.Namespace = "default"

	pm.RuntimeMock.On("BaseConfig").Return(pc)

	s.On("ListReleases", mock.Anything).Return([]*models.Release{dep}, nil)

	return dep
}

func TestAddsAnnotationToDeploymentWhenReleaseFound(t *testing.T) {
	h, s, pp, pm := setupWebhook(t)
	setupReleases(t, pp, pm, s, "api-deployment")

	data := testutils.GetTestData(t, "admission_review.json")
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))

	h(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)

	// get the
	resp := map[string]interface{}{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	patch, _ := base64.StdEncoding.DecodeString(resp["response"].(map[string]interface{})["patch"].(string))

	require.Contains(t, string(patch), "consul-releaser")
}

func TestCallsRuntimeDeployWhenReleaseFound(t *testing.T) {
	h, s, pp, pm := setupWebhook(t)
	setupReleases(t, pp, pm, s, "api-deployment")

	data := testutils.GetTestData(t, "admission_review.json")
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))

	h(rr, r)

	require.Eventually(t, func() bool {
		calls := pm.RuntimeMock.Calls
		for _, c := range calls {
			if c.Method == "InitPrimary" {
				return true
			}
		}

		return false
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestDoesNothingWhenNoReleaseFound(t *testing.T) {
	h, s, pp, pm := setupWebhook(t)
	setupReleases(t, pp, pm, s, "not-found")

	data := testutils.GetTestData(t, "admission_review.json")
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))

	h(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)

	// get the
	resp := map[string]interface{}{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	patch, _ := base64.StdEncoding.DecodeString(resp["response"].(map[string]interface{})["patch"].(string))

	require.NotContains(t, string(patch), "consul-releaser")
}
