package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	appmetrics "github.com/nicholasjackson/consul-release-controller/metrics"
	"github.com/nicholasjackson/consul-release-controller/models"
	plugins "github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/state"
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
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Get handler lists current deployments
func (d *ReleaseHandler) GetAll(rw http.ResponseWriter, req *http.Request) {
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
		resp = append(resp, GetResponse{Name: dep.Name, Status: dep.CurrentState, Version: dep.Version})
	}

	json.NewEncoder(rw).Encode(resp)
	mFinal(http.StatusOK)
}

func (d *ReleaseHandler) GetSingle(rw http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")

	d.logger.Info("Release GET handler called")
	mFinal := d.metrics.HandleRequest("release/get", nil)

	rel, err := d.store.GetRelease(name)

	if err == state.ReleaseNotFound {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(rw).Encode(rel)
	mFinal(http.StatusOK)
}

// Delete handler deletes a deployment
func (d *ReleaseHandler) Delete(rw http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")

	d.logger.Info("Release DELETE handler called", "name", name)
	mFinal := d.metrics.HandleRequest("deployment/delete", nil)

	rel, err := d.store.GetRelease(name)

	if err == state.ReleaseNotFound {
		d.logger.Error("unable to find release, not found", "name", name)
		mFinal(http.StatusNotFound)

		http.Error(rw, fmt.Sprintf("release %s not found", name), http.StatusNotFound)
		return
	}

	if err != nil {
		d.logger.Error("unable to get release", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	// cleanup any config
	err = rel.Destroy()
	if err != nil {
		d.logger.Error("unable to cleanup config", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	err = d.store.DeleteRelease(name)
	if err != nil {
		d.logger.Error("unable to delete release", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	mFinal(http.StatusOK)
}
