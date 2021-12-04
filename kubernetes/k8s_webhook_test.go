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
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/plugins/kubernetes"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupWebhook(t *testing.T) (func(w http.ResponseWriter, r *http.Request), *state.MockStore, *plugins.Mocks) {
	pp, pm := plugins.BuildMocks(t)
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}

	wh, _ := NewK8sWebhook(l, m, s, pp)

	return wh.Mutating(), s, pm
}

func setupReleases(t *testing.T, pm *plugins.Mocks, s *state.MockStore, name string) *models.Release {
	depData := testutils.GetTestData(t, "idle_kubernetes_deployment.json")

	dep := &models.Release{}
	dep.FromJsonBody(ioutil.NopCloser(bytes.NewBuffer(depData)))

	pm.RuntimeMock.On("GetConfig").Return(&kubernetes.PluginConfig{Deployment: name})
	s.On("ListReleases", mock.Anything).Return([]*models.Release{dep}, nil)

	return dep
}

func TestAddsAnnotationWhenDeploymentFound(t *testing.T) {
	h, s, pm := setupWebhook(t)
	setupReleases(t, pm, s, "api-deployment")

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

func TestCallsDeployWhenDeploymentFound(t *testing.T) {
	h, s, pm := setupWebhook(t)
	release := setupReleases(t, pm, s, "api-deployment")

	data := testutils.GetTestData(t, "admission_review.json")
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))

	h(rr, r)

	assert.Eventually(t, func() bool {
		return release.StateIs(models.StateMonitor)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestDoesNothingWhenNoDeploymentFound(t *testing.T) {
	h, s, pm := setupWebhook(t)
	setupReleases(t, pm, s, "not-found")

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
