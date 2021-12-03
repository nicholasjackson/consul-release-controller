package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/testutils"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDeployment(t *testing.T) (*Deployment, *httptest.ResponseRecorder, *state.MockStore, *plugins.ProviderMock) {
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}

	pp, _ := plugins.BuildMocks(t)

	rw := httptest.NewRecorder()

	return NewDeployment(l, m, s, pp), rw, s, pp
}

func TestDeploymentPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw, _, _ := setupDeployment(t)
	r := httptest.NewRequest("POST", "/", nil)

	d.Post(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestDeploymentPostWithStoreErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(fmt.Errorf("boom"))

	td := testutils.GetTestData(t, "valid_kubernetes_deployment.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentPostWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(nil)

	td := testutils.GetTestData(t, "valid_kubernetes_deployment.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestDeploymentPostCallsConfigure(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(nil)

	td := testutils.GetTestData(t, "valid_kubernetes_deployment.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	// work is done in the background check t
	assert.Eventually(t, func() bool {
		d := s.Calls[0].Arguments[0].(*models.Deployment)
		return d.StateIs(models.StateIdle)
	}, 100*time.Millisecond, 1*time.Millisecond)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestDeploymentGetWithErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)

	s.On("ListDeployments").Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("GET", "/", nil)

	d.Get(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentGetReturnsStatus(t *testing.T) {
	d, rw, s, pp := setupDeployment(t)

	m1 := &models.Deployment{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Build(pp)

	m2 := &models.Deployment{}
	m2.Name = "test2"
	m2.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Build(pp)

	deps := []*models.Deployment{m1, m2}

	s.On("ListDeployments").Return(deps, nil)

	r := httptest.NewRequest("GET", "/", nil)
	d.Get(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	resp := []GetResponse{}
	json.Unmarshal(rw.Body.Bytes(), &resp)

	assert.Equal(t, "test1", resp[0].Name)
	assert.Equal(t, "state_start", resp[0].Status)

	assert.Equal(t, "test2", resp[1].Name)
	assert.Equal(t, "state_start", resp[1].Status)
}

func TestDeploymentDeleteWithErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)

	s.On("DeleteDeployment", "consul").Return(fmt.Errorf("boom"))

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentDeleteWithNotFoundReturns404(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)

	s.On("DeleteDeployment", "consul").Return(state.DeploymentNotFound)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestDeploymentDeleteWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, _ := setupDeployment(t)
	s.On("DeleteDeployment", "consul").Return(nil)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))

	d.Delete(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}
