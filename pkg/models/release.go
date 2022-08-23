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

	Statehistory []StateHistory `json:"state_history"`
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
	return d.Statehistory
}

func (d *Release) UpdateState(state string) {
	if d.Statehistory == nil {
		d.Statehistory = []StateHistory{}
	}

	d.Statehistory = append(d.Statehistory, StateHistory{Time: time.Now(), State: state})

	// ensure the state history never grows beyond 50 items
	if len(d.Statehistory) > 50 {
		d.Statehistory = d.Statehistory[len(d.Statehistory)-50:]
	}
}

func (d *Release) CurrentState() string {
	if len(d.Statehistory) == 0 {
		return ""
	}

	return d.Statehistory[len(d.Statehistory)-1].State
}
