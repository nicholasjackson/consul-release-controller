package api

import (
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/state"
)

type Deployment struct {
	Logger hclog.Logger
	Store  state.Store
}

func NewDeployment(logger hclog.Logger, store state.Store) *Deployment {
	return &Deployment{Logger: logger, Store: store}
}

func (d *Deployment) Post(rw http.ResponseWriter, req *http.Request) {
	dep := &models.Deployment{}
	err := dep.FromJsonBody(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// store the new deployment
	err = d.Store.UpsertDeployment(*dep)
	if err != nil {
		d.Logger.Error("unable to upsert deployment", "deployment", dep, "error", err)

		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
