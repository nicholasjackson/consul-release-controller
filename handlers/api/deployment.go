package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/models"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/state"
)

type Deployment struct {
	Logger          hclog.Logger
	Store           state.Store
	Metrics         metrics.Metrics
	PluginProviders plugins.Provider
}

func NewDeployment(l hclog.Logger, m metrics.Metrics, s state.Store, p plugins.Provider) *Deployment {
	return &Deployment{Logger: l, Metrics: m, Store: s, PluginProviders: p}
}

// Post handler creates a new deployment
func (d *Deployment) Post(rw http.ResponseWriter, req *http.Request) {
	d.Logger.Info("Deployment POST handler called")
	mFinal := d.Metrics.HandleRequest("deployment/post", nil)

	dep := &models.Deployment{}
	err := dep.FromJsonBody(req.Body)
	if err != nil {
		d.Logger.Error("unable to upsert deployment", "deployment", *dep, "error", err)
		mFinal(http.StatusBadRequest)

		http.Error(rw, "invalid request body", http.StatusBadRequest)
		return
	}

	err = dep.Build(d.PluginProviders)
	if err != nil {
		d.Logger.Error("unable to build deployment", "deployment", *dep, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to save deployment", http.StatusInternalServerError)
	}

	// store the new deployment
	err = d.Store.UpsertDeployment(dep)
	if err != nil {
		d.Logger.Error("unable to upsert deployment", "deployment", *dep, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to save deployment", http.StatusInternalServerError)
		return
	}

	// move the state to initialize
	dep.Configure()

	mFinal(http.StatusOK)
}

type GetResponse struct {
	Name   string
	Status string
}

// Get handler lists current deployments
func (d *Deployment) Get(rw http.ResponseWriter, req *http.Request) {
	d.Logger.Info("Deployment GET handler called")
	mFinal := d.Metrics.HandleRequest("deployment/get", nil)

	deps, err := d.Store.ListDeployments()
	if err != nil {
		d.Logger.Error("unable to list deployments", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to fetch deployments", http.StatusInternalServerError)
		return
	}

	resp := []GetResponse{}
	for _, dep := range deps {
		resp = append(resp, GetResponse{Name: dep.Name, Status: dep.State()})
	}

	json.NewEncoder(rw).Encode(resp)
	mFinal(http.StatusOK)
}

// Delete handler deletes a deployment
func (d *Deployment) Delete(rw http.ResponseWriter, req *http.Request) {
	name := req.Context().Value("name").(string)

	d.Logger.Info("Deployment DELETE handler called", "name", name)
	mFinal := d.Metrics.HandleRequest("deployment/delete", nil)

	err := d.Store.DeleteDeployment(name)

	if err == state.DeploymentNotFound {
		d.Logger.Error("unable to delete deployment, not found", "name", name)
		mFinal(http.StatusNotFound)

		http.Error(rw, fmt.Sprintf("deployment %s not found", name), http.StatusNotFound)
		return
	}

	if err != nil {
		d.Logger.Error("unable to delete deployment", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete deployment", http.StatusInternalServerError)
		return
	}

	mFinal(http.StatusOK)
}
