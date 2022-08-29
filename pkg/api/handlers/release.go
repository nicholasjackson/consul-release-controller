package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

// ReleaseHandler handles the CRUD operations for releases
type ReleaseHandler struct {
	logger          hclog.Logger
	store           interfaces.Store
	metrics         interfaces.Metrics
	pluginProviders interfaces.Provider
}

// NewReleaseHandler creates a new ReleaseHandler
func NewReleaseHandler(p interfaces.Provider) *ReleaseHandler {
	return &ReleaseHandler{logger: p.GetLogger().Named("release_handler"), metrics: p.GetMetrics(), store: p.GetDataStore(), pluginProviders: p}
}

// Post handler for creating and updating releases
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

// GetAllResponse is returned by GetAll requests
type GetAllResponse struct {
	Name                 string `json:"name"`
	Status               string `json:"status"`
	LastDeploymentStatus string `json:"last_deployment_status"`
	CandidateTraffic     int    `json:"candidate_traffic"`
	Version              string `json:"version"`
}

// GetAll handler returns all current releases
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
			rh.logger.Error("Unable to get statemachine for", "release", rel.Name)
		} else {
			s = sm.CurrentState()
		}

		var traffic float64

		deploymentStatus := ""

		// get the last strategy status
		d, err := rh.pluginProviders.GetDataStore().CreatePluginStateStore(rel, "strategy").GetState()
		if err == nil {
			status := map[string]interface{}{}
			err := json.Unmarshal(d, &status)
			if err != nil {
				rh.logger.Error("Unable to marshal status from strategy", "error", err)
			}

			if ct, ok := status["candidate_traffic"].(float64); ok {
				traffic = ct
			}

			if s, ok := status["status"].(string); ok {
				deploymentStatus = s
			}
		}

		resp = append(resp, GetAllResponse{Name: rel.Name, Status: s, Version: rel.Version, CandidateTraffic: int(traffic), LastDeploymentStatus: deploymentStatus})
	}

	json.NewEncoder(rw).Encode(resp)
	mFinal(http.StatusOK)
}

// GetSingleResponse returns a single release
type GetSingleResponse struct {
	models.Release
	CurrentState string                `json:"current_state"`
	StateHistory []models.StateHistory `json:"state_history"`
}

// GetSingle handler returns a release related to the "name" HTTP querystring parameter
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
	gsr.CurrentState = rel.CurrentState()
	gsr.StateHistory = rel.StateHistory()

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
		rh.logger.Error("Unable to get state machine for", "release", rel.Name)
		mFinal(http.StatusInternalServerError)

		http.Error(rw, "unable to find state machine for release", http.StatusInternalServerError)
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

	// wait until finished
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		for {
			if ctx.Err() != nil {
				rh.logger.Error("Timeout waiting to destroy release", "name", rel.Name)
				return
			}

			// destroy is complete
			if sm.CurrentState() == interfaces.StateIdle {
				rh.logger.Info("Destroy complete, removing release", "name", rel.Name)
				rh.pluginProviders.DeleteStateMachine(rel)
				if err != nil {
					rh.logger.Error("Unable to delete state machine", "name", rel.Name, "error", err)
				}

				err = rh.pluginProviders.GetDataStore().DeleteRelease(rel.Name)
				if err != nil {
					rh.logger.Error("Unable to delete release", "name", rel.Name, "error", err)
				}

				return
			}

			if sm.CurrentState() == interfaces.StateFail {
				rh.logger.Error("Unable to destroy release", "name", rel.Name)
				return
			}

			rh.logger.Info("Waiting for destroy to complete", "current_state", sm.CurrentState())
			time.Sleep(2 * time.Second)
		}
	}()

	mFinal(http.StatusOK)
}
