package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nicholasjackson/consul-release-controller/metrics"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupRelease(t *testing.T) (http.Handler, *httptest.ResponseRecorder, *state.MockStore, *mocks.ProviderMock) {
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}

	pp, _ := mocks.BuildMocks(t)

	rw := httptest.NewRecorder()
	apiHandler := NewReleaseHandler(l, m, s, pp)

	rtr := chi.NewRouter()

	// configure the main API
	rtr.Post("/v1/releases", apiHandler.Post)
	rtr.Get("/v1/releases", apiHandler.GetAll)
	rtr.Get("/v1/releases/{name}", apiHandler.GetSingle)
	rtr.Delete("/v1/releases/{name}", apiHandler.Delete)

	return rtr, rw, s, pp
}

func TestReleaseHandlerPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw, _, _ := setupRelease(t)
	r := httptest.NewRequest("POST", "/v1/releases", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestReleaseHandlerPostWithStoreErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("UpsertRelease", mock.Anything).Return(fmt.Errorf("boom"))

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/v1/releases", bytes.NewBuffer(td))

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerPostWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("UpsertRelease", mock.Anything).Return(nil)

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/v1/releases", bytes.NewBuffer(td))

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestReleaseHandlerGetWithErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("ListReleases", mock.Anything).Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("GET", "/v1/releases", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerGetReturnsStatus(t *testing.T) {
	d, rw, s, pp := setupRelease(t)

	m1 := &models.Release{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Build(pp)

	m2 := &models.Release{}
	m2.Name = "test2"
	m2.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Build(pp)

	deps := []*models.Release{m1, m2}

	s.On("ListReleases", mock.Anything).Return(deps, nil)

	r := httptest.NewRequest("GET", "/v1/releases", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	resp := []GetResponse{}
	json.Unmarshal(rw.Body.Bytes(), &resp)

	assert.Equal(t, "test1", resp[0].Name)
	assert.Equal(t, "state_start", resp[0].Status)

	assert.Equal(t, "test2", resp[1].Name)
	assert.Equal(t, "state_start", resp[1].Status)
}

func TestReleaseHandlerGetSingleReturnsReleaseWhenExists(t *testing.T) {
	d, rw, s, pp := setupRelease(t)

	m1 := &models.Release{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Build(pp)

	s.On("GetRelease", mock.Anything).Return(m1, nil)

	r := httptest.NewRequest("GET", "/v1/releases/test1", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	rel := models.Release{}
	err := json.NewDecoder(rw.Body).Decode(&rel)

	require.NoError(t, err)
	require.Equal(t, m1.Name, rel.Name)
}

func TestReleaseHandlerGetSingleReturns404WhenNotFound(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("GetRelease", mock.Anything).Return(nil, state.ReleaseNotFound)

	r := httptest.NewRequest("GET", "/v1/releases/test1", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestReleaseHandlerDeleteWithGetErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("GetRelease", "consul").Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerDeleteWithNotFoundReturns404(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("GetRelease", "consul").Return(nil, state.ReleaseNotFound)

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestReleaseHandlerDeleteWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, pp := setupRelease(t)

	m := &models.Release{}
	m.Name = "test2"
	m.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m.CurrentState = models.StateIdle
	m.Build(pp)

	s.On("GetRelease", "consul").Return(m, nil)
	s.On("DeleteRelease", "consul").Return(nil)

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}
