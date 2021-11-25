package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupDeployment(t *testing.T) (*Deployment, *httptest.ResponseRecorder, *state.MockStore) {
	logger := hclog.Default()
	store := &state.MockStore{}

	rw := httptest.NewRecorder()

	return NewDeployment(logger, store), rw, store
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

var exampleDeployment = models.Deployment{
	ConsulService: "payments",
}
