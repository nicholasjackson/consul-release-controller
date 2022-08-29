package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"

	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupRelease(t *testing.T) (http.Handler, *httptest.ResponseRecorder, *mocks.ProviderMock, *mocks.Mocks) {
	pp, m := mocks.BuildMocks(t)

	rw := httptest.NewRecorder()
	apiHandler := NewReleaseHandler(pp)

	rtr := chi.NewRouter()

	// configure the main API
	rtr.Post("/v1/releases", apiHandler.Post)
	rtr.Get("/v1/releases", apiHandler.GetAll)
	rtr.Get("/v1/releases/{name}", apiHandler.GetSingle)
	rtr.Delete("/v1/releases/{name}", apiHandler.Delete)

	return rtr, rw, pp, m
}

func TestReleaseHandlerPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw, _, _ := setupRelease(t)
	r := httptest.NewRequest("POST", "/v1/releases", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestReleaseHandlerPostWithStoreErrorReturnsError(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "UpsertRelease")
	m.StoreMock.On("UpsertRelease", mock.Anything).Return(fmt.Errorf("boom"))

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/v1/releases", bytes.NewBuffer(td))

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerPostWithNoErrorReturnsOk(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/v1/releases", bytes.NewBuffer(td))

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
	m.StoreMock.AssertCalled(t, "UpsertRelease", mock.Anything)
}

func TestReleaseHandlerGetWithErrorReturnsError(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "ListReleases")
	m.StoreMock.On("ListReleases", mock.Anything).Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("GET", "/v1/releases", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerGetAllReturnsStatus(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	m1 := &models.Release{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}

	m2 := &models.Release{}
	m2.Name = "test2"
	m2.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}

	releases := []*models.Release{m1, m2}

	testutils.ClearMockCall(&m.StoreMock.Mock, "ListReleases")
	m.StoreMock.On("ListReleases", mock.Anything).Return(releases, nil)

	testutils.ClearMockCall(&m.StateMachineMock.Mock, "CurrentState")
	m.StateMachineMock.On("CurrentState").Once().Return(interfaces.StateStart)
	m.StateMachineMock.On("CurrentState").Once().Return(interfaces.StateIdle)

	r := httptest.NewRequest("GET", "/v1/releases", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	resp := []GetAllResponse{}
	json.Unmarshal(rw.Body.Bytes(), &resp)

	assert.Equal(t, releases[0].Name, resp[0].Name)
	assert.Equal(t, interfaces.StateStart, resp[0].Status)

	assert.Equal(t, releases[1].Name, resp[1].Name)
	assert.Equal(t, interfaces.StateIdle, resp[1].Status)
}

func TestReleaseHandlerGetSingleReturnsReleaseWhenExists(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	m1 := &models.Release{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Monitor = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Strategy = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetRelease")
	m.StoreMock.On("GetRelease", "test1").Return(m1, nil)

	r := httptest.NewRequest("GET", "/v1/releases/test1", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	rel := models.Release{}
	err := json.NewDecoder(rw.Body).Decode(&rel)

	require.NoError(t, err)
	require.Equal(t, m1.Name, rel.Name)
}

func TestReleaseHandlerGetSingleReturns404WhenNotFound(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetRelease")
	m.StoreMock.On("GetRelease", mock.Anything).Return(nil, interfaces.ReleaseNotFound)

	r := httptest.NewRequest("GET", "/v1/releases/test1", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestReleaseHandlerDeleteWithGetErrorReturnsError(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetRelease")
	m.StoreMock.On("GetRelease", "consul").Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerDeleteWithNotFoundReturns404(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetRelease")
	m.StoreMock.On("GetRelease", mock.Anything).Return(nil, interfaces.ReleaseNotFound)

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)
	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestReleaseHandlerDeleteWithNoErrorReturnsOk(t *testing.T) {
	d, rw, _, m := setupRelease(t)

	rel := &models.Release{}
	rel.Name = "test2"

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetRelease")
	testutils.ClearMockCall(&m.StoreMock.Mock, "DeleteRelease")
	m.StoreMock.On("GetRelease", "consul").Return(m, nil)
	m.StoreMock.On("DeleteRelease", "consul").Return(nil)

	r := httptest.NewRequest("DELETE", "/v1/releases/consul", nil)

	d.ServeHTTP(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}
