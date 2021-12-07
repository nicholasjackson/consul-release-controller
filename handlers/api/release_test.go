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

func setupRelease(t *testing.T) (*ReleaseHandler, *httptest.ResponseRecorder, *state.MockStore, *plugins.ProviderMock) {
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}

	pp, _ := plugins.BuildMocks(t)

	rw := httptest.NewRecorder()

	return NewReleaseHandler(l, m, s, pp), rw, s, pp
}

func TestReleaseHandlerPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw, _, _ := setupRelease(t)
	r := httptest.NewRequest("POST", "/", nil)

	d.Post(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestReleaseHandlerPostWithStoreErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("UpsertRelease", mock.Anything).Return(fmt.Errorf("boom"))

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerPostWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("UpsertRelease", mock.Anything).Return(nil)

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestReleaseHandlerPostCallsConfigure(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("UpsertRelease", mock.Anything).Return(nil)

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(td))

	d.Post(rw, r)

	// work is done in the background check t
	assert.Eventually(t, func() bool {
		d := s.Calls[0].Arguments[0].(*models.Release)
		return d.StateIs(models.StateIdle)
	}, 100*time.Millisecond, 1*time.Millisecond)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestReleaseHandlerGetWithErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("ListReleases", mock.Anything).Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("GET", "/", nil)

	d.Get(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerGetReturnsStatus(t *testing.T) {
	d, rw, s, pp := setupRelease(t)

	m1 := &models.Release{}
	m1.Name = "test1"
	m1.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m1.Build(pp)

	m2 := &models.Release{}
	m2.Name = "test2"
	m2.Releaser = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Runtime = &models.PluginConfig{Name: "test", Config: []byte(`{}`)}
	m2.Build(pp)

	deps := []*models.Release{m1, m2}

	s.On("ListReleases", mock.Anything).Return(deps, nil)

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

func TestReleaseHandlerDeleteWithErrorReturnsError(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("DeleteRelease", "consul").Return(fmt.Errorf("boom"))

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestReleaseHandlerDeleteWithNotFoundReturns404(t *testing.T) {
	d, rw, s, _ := setupRelease(t)

	s.On("DeleteRelease", "consul").Return(state.ReleaseNotFound)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestReleaseHandlerDeleteWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s, _ := setupRelease(t)
	s.On("DeleteRelease", "consul").Return(nil)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))

	d.Delete(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}
