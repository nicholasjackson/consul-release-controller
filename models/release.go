package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Release struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	Version string `json:"version"`

	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`

	Releaser           *PluginConfig   `json:"releaser"`
	Runtime            *PluginConfig   `json:"runtime"`
	Strategy           *PluginConfig   `json:"strategy"`
	Monitor            *PluginConfig   `json:"monitor"`
	Webhooks           []*PluginConfig `json:"webhooks"`
	PostDeploymentTest *PluginConfig   `json:"post_deployment_test"`

	stateHistory []StateHistory
}

// StateHistory is a struct that defines the state at a point in time
type StateHistory struct {
	Time  time.Time `json:"time"`
	State string    `json:"state"`
}

type PluginConfig struct {
	Name   string          `json:"plugin_name"`
	Config json.RawMessage `json:"config"`
}

// FromJsonBody decodes the json body into the Deployment type
func (d *Release) FromJsonBody(r io.ReadCloser) error {
	if r == nil {
		return fmt.Errorf("no json body provided")
	}

	return json.NewDecoder(r).Decode(d)
}

// ToJson serializes the deployment to json
func (d *Release) ToJson() []byte {
	data, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return data
}

// StateHistory returns all the states for the statemachine
func (d *Release) StateHistory() []StateHistory {
	return d.stateHistory
}

func (d *Release) UpdateState(state string) {
	if d.stateHistory == nil {
		d.stateHistory = []StateHistory{}
	}

	d.stateHistory = append(d.stateHistory, StateHistory{Time: time.Now(), State: state})
}

func (d *Release) CurrentState() string {
	if len(d.stateHistory) == 0 {
		return ""
	}

	fmt.Println("history", d.stateHistory)

	return d.stateHistory[len(d.stateHistory)-1].State
}
