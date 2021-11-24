package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/state"
	"github.com/stretchr/testify/assert"
)

func setupDeployment(t *testing.T) (*Deployment, *httptest.ResponseRecorder) {
	logger := hclog.Default()
	store := &state.MockStore{}

	rw := httptest.NewRecorder()

	return NewDeployment(logger, store), rw
}

func TestDeploymentPostWithInvalidBodyReturnsBadRequest(t *testing.T) {
	d, rw := setupDeployment(t)
	r := httptest.NewRequest("POST", "/", nil)

	d.Post(rw, r)

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}
