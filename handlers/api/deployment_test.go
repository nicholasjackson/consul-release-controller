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

	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/metrics"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDeployment(t *testing.T) (*Deployment, *httptest.ResponseRecorder, *state.MockStore, *clients.Clients) {
	l := hclog.Default()
	s := &state.MockStore{}
	m := &metrics.Null{}
	c := &clients.Clients{
		Consul: &clients.MockConsul{},
	}

	rw := httptest.NewRecorder()

	return NewDeployment(l, m, s, c), rw, s, c
}

func TestDeploymentPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw, _ := setupDeployment(t)
	r := httptest.NewRequest("POST", "/", nil)

	d.Post(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestDeploymentPostWithStoreErrorReturnsError(t *testing.T) {
	d, rw, s := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(fmt.Errorf("boom"))

	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(exampleDeployment.ToJson()))

	d.Post(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentPostWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(nil)

	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(exampleDeployment.ToJson()))

	d.Post(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}

func TestDeploymentPostCallsInitialize(t *testing.T) {
	d, rw, s, c := setupDeployment(t)
	s.On("UpsertDeployment", mock.Anything).Return(nil)

	r := httptest.NewRequest("POST", "/", bytes.NewBuffer(exampleDeployment.ToJson()))

	d.Post(rw, r)

	// work is done in the background check t
	assert.Eventually(t, func() bool {
		return false
	}, 100*time.Millisecond, 1*time.Millisecond)

	assert.Equal(t, http.StatusOK, rw.Code)
}

var exampleDeployment = models.Deployment{
	ConsulService: "payments",
}

func TestDeploymentGetWithErrorReturnsError(t *testing.T) {
	d, rw, s := setupDeployment(t)

	s.On("ListDeployments").Return(nil, fmt.Errorf("boom"))

	r := httptest.NewRequest("GET", "/", nil)
	d.Get(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentGetReturnsStatus(t *testing.T) {
	d, rw, s := setupDeployment(t)

	m1 := models.NewDeployment(nil, nil, nil)
	m1.ConsulService = "test1"
	m2 := models.NewDeployment(nil, nil, nil)
	m2.ConsulService = "test2"
	deps := []*models.Deployment{m1, m2}

	s.On("ListDeployments").Return(deps, nil)

	r := httptest.NewRequest("GET", "/", nil)
	d.Get(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)

	resp := []GetResponse{}
	json.Unmarshal(rw.Body.Bytes(), &resp)

	assert.Equal(t, "test1", resp[0].Name)
	assert.Equal(t, "inactive", resp[0].Status)

	assert.Equal(t, "test2", resp[1].Name)
	assert.Equal(t, "inactive", resp[1].Status)
}

func TestDeploymentDeleteWithErrorReturnsError(t *testing.T) {
	d, rw, s := setupDeployment(t)

	s.On("DeleteDeployment", "consul").Return(fmt.Errorf("boom"))

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusInternalServerError, rw.Code)
}

func TestDeploymentDeleteWithNotFoundReturns404(t *testing.T) {
	d, rw, s := setupDeployment(t)

	s.On("DeleteDeployment", "consul").Return(state.DeploymentNotFound)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))
	d.Delete(rw, r)

	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestDeploymentDeleteWithNoErrorReturnsOk(t *testing.T) {
	d, rw, s := setupDeployment(t)
	s.On("DeleteDeployment", "consul").Return(nil)

	r := httptest.NewRequest("DELETE", "/consul", nil)
	r = r.WithContext(context.WithValue(context.Background(), "name", "consul"))

	d.Delete(rw, r)

	assert.Equal(t, http.StatusOK, rw.Code)
}
