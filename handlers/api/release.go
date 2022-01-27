package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type ReleaseHandler struct {
	logger          hclog.Logger
	store           interfaces.Store
	metrics         interfaces.Metrics
	pluginProviders interfaces.Provider
}

func NewReleaseHandler(l hclog.Logger, p interfaces.Provider) *ReleaseHandler {
	return &ReleaseHandler{logger: l, metrics: p.GetMetrics(), store: p.GetDataStore(), pluginProviders: p}
}

// Post handler creates a new deployment
func (rh *ReleaseHandler) Post(rw http.ResponseWriter, req *http.Request) {
	rh.logger.Info("Release POST handler called")
	mFinal := rh.metrics.HandleRequest("release_handler", map[string]string{"method": "post"})

	rel := &models.Release{}
	err := rel.FromJsonBody(req.Body)
	if err != nil {
		rh.logger.Error("unable to upsert release", "release", *rel, "error", err)
		mFinal(http.StatusBadRequest)

		http.Error(rw, "invalid request body", http.StatusBadRequest)
		return
	}

	// store the new deployment
	err = rh.store.UpsertRelease(rel)
	if err != nil {
		rh.logger.Error("unable to upsert release", "deployment", *rel, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to save release", http.StatusInternalServerError)
		return
	}

	// get a statemachine for the release
	sm, err := rh.pluginProviders.GetStateMachine(rel)
	if err != nil {
		rh.logger.Error("unable to create statemachine for the release", "release", *rel, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to create statemachine", http.StatusInternalServerError)
		return
	}

	// trigger the configuration of the config
	err = sm.Configure()
	if err != nil {
		rh.logger.Error("unable to configure release", "release", *rel, "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to configure release", http.StatusInternalServerError)
	}

	mFinal(http.StatusOK)
}

type GetAllResponse struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Get handler lists current deployments
func (rh *ReleaseHandler) GetAll(rw http.ResponseWriter, req *http.Request) {
	rh.logger.Info("Release GET handler called")
	mFinal := rh.metrics.HandleRequest("release_handler", map[string]string{"method": "get_all"})

	releases, err := rh.store.ListReleases(nil)
	if err != nil {
		rh.logger.Error("unable to list releases", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to fetch releases", http.StatusInternalServerError)
		return
	}

	resp := []GetAllResponse{}
	for _, rel := range releases {
		s := "unknown"
		sm, err := rh.pluginProviders.GetStateMachine(rel)
		if err != nil {
			rh.logger.Error("Unaable to get statemachine for", "release", rel.Name)
		} else {
			s = sm.CurrentState()
		}

		resp = append(resp, GetAllResponse{Name: rel.Name, Status: s, Version: rel.Version})
	}

	json.NewEncoder(rw).Encode(resp)
	mFinal(http.StatusOK)
}

type GetSingleResponse struct {
	models.Release
	CurrentState string                    `json:"current_state"`
	StateHistory []interfaces.StateHistory `json:"state_history"`
}

func (rh *ReleaseHandler) GetSingle(rw http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")

	rh.logger.Info("Release GET single handler called")
	mFinal := rh.metrics.HandleRequest("release_handler", map[string]string{"method": "get_single"})

	rel, err := rh.store.GetRelease(name)

	if err == interfaces.ReleaseNotFound {
		rw.WriteHeader(http.StatusNotFound)
		mFinal(http.StatusNotFound)
		return
	}

	gsr := GetSingleResponse{}
	gsr.Release = *rel

	// add the state history
	sm, err := rh.pluginProviders.GetStateMachine(rel)
	if err != nil {
		rh.logger.Error("Unaable to get statemachine for", "release", rel.Name)
	} else {
		gsr.CurrentState = sm.CurrentState()
		gsr.StateHistory = sm.StateHistory()
	}

	json.NewEncoder(rw).Encode(&gsr)
	mFinal(http.StatusOK)
}

// Delete handler deletes a deployment
func (rh *ReleaseHandler) Delete(rw http.ResponseWriter, req *http.Request) {
	name := chi.URLParam(req, "name")

	rh.logger.Info("Release DELETE handler called", "name", name)
	mFinal := rh.metrics.HandleRequest("release_handler", map[string]string{"method": "delete"})

	rel, err := rh.store.GetRelease(name)

	if err == interfaces.ReleaseNotFound {
		rh.logger.Error("unable to find release, not found", "name", name)
		mFinal(http.StatusNotFound)

		http.Error(rw, fmt.Sprintf("release %s not found", name), http.StatusNotFound)
		return
	}

	if err != nil {
		rh.logger.Error("unable to get release", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	sm, err := rh.pluginProviders.GetStateMachine(rel)
	if err != nil {
		rh.logger.Error("Unaable to get statemachine for", "release", rel.Name)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to find statemachine for release", http.StatusInternalServerError)
		return
	}

	// cleanup any config
	err = sm.Destroy()
	if err != nil {
		rh.logger.Error("unable to cleanup config", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to cleanup release", http.StatusInternalServerError)
		return
	}

	// remove the statemachine
	rh.pluginProviders.DeleteStateMachine(rel)

	// delete the release
	err = rh.store.DeleteRelease(name)
	if err != nil {
		rh.logger.Error("unable to delete release", "error", err)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to delete release", http.StatusInternalServerError)
		return
	}

	mFinal(http.StatusOK)
}
