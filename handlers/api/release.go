package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-hclog"
	appmetrics "github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/models"
	plugins "github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/state"
)

type ReleaseHandler struct {
	logger          hclog.Logger
	store           state.Store
	metrics         appmetrics.Metrics
	pluginProviders plugins.Provider
}

func NewReleaseHandler(l hclog.Logger, m appmetrics.Metrics, s state.Store, p plugins.Provider) *ReleaseHandler {
	return &ReleaseHandler{logger: l, metrics: m, store: s, pluginProviders: p}
}

// Post handler creates a new deployment
func (d *ReleaseHandler) Post(rw http.ResponseWriter, req *http.Request) {
	d.logger.Info("Release POST handler called")
	mFinal := d.metrics.HandleRequest("release/post", nil)

	dep := &models.Release{}
	err := dep.FromJsonBody(req.Body)
	if err != nil {
		d.logger.Error("unable to upsert release", "release", *dep, "error", err)
		mFinal(http.StatusBadRequest)

		http.Error(rw, "invalid request body", http.StatusBadRequest)
		return
	}

	err = dep.Build(d.pluginProviders)
	if err != nil {
		d.logger.Error("unable to build release", "deployment", *dep, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to save release", http.StatusInternalServerError)
	}

	// store the new deployment
	err = d.store.UpsertRelease(dep)
	if err != nil {
		d.logger.Error("unable to upsert release", "deployment", *dep, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to save release", http.StatusInternalServerError)
		return
	}

	// trigger the configuration of the config
	err = dep.Configure()
	if err != nil {
		d.logger.Error("unable to configure release", "deployment", *dep, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to configure release", http.StatusInternalServerError)
	}

	mFinal(http.StatusOK)
}

type GetResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Get handler lists current deployments
func (d *ReleaseHandler) Get(rw http.ResponseWriter, req *http.Request) {
	d.logger.Info("Release GET handler called")
	mFinal := d.metrics.HandleRequest("release/get", nil)

	deps, err := d.store.ListReleases(nil)
	if err != nil {
		d.logger.Error("unable to list releases", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to fetch releases", http.StatusInternalServerError)
		return
	}

	resp := []GetResponse{}
	for _, dep := range deps {
		resp = append(resp, GetResponse{Name: dep.Name, Status: dep.CurrentState})
	}

	json.NewEncoder(rw).Encode(resp)
	mFinal(http.StatusOK)
}

// Delete handler deletes a deployment
func (d *ReleaseHandler) Delete(rw http.ResponseWriter, req *http.Request) {
	name := req.Context().Value("name").(string)

	d.logger.Info("Release DELETE handler called", "name", name)
	mFinal := d.metrics.HandleRequest("deployment/delete", nil)

	err := d.store.DeleteRelease(name)

	if err == state.ReleaseNotFound {
		d.logger.Error("unable to delete release, not found", "name", name)
		mFinal(http.StatusNotFound)

		http.Error(rw, fmt.Sprintf("release %s not found", name), http.StatusNotFound)
		return
	}

	if err != nil {
		d.logger.Error("unable to delete release", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	mFinal(http.StatusOK)
}
