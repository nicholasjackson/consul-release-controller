package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
)

type Deployment struct {
	Active    bool
	StartTime time.Time
	EndTime   time.Time

	Name string `json:"name"`

	Releaser *PluginConfig `json:"releaser"`
	Runtime  *PluginConfig `json:"runtime"`

	state *fsm.FSM
}

type PluginConfig struct {
	Name   string          `json:"plugin_name"`
	Config json.RawMessage `json:"config"`
}

// Build creates a new deployment setting the state to inactive
func (d *Deployment) Build(pluginProvider plugins.Provider) error {
	// configure the setup plugin
	sp, err := pluginProvider.CreateReleaser(d.Releaser.Name)
	if err != nil {
		return err
	}

	// configure the releaser plugin
	sp.Configure(d.Releaser.Config)

	// configure the runtime plugin
	rp, err := pluginProvider.CreateRuntime(d.Runtime.Name)
	if err != nil {
		return err
	}

	// configure the runtime plugin
	rp.Configure(d.Runtime.Config)

	fsm := newFSM(d, sp, rp)
	d.state = fsm

	return err
}

// FromJsonBody decodes the json body into the Deployment type
func (d *Deployment) FromJsonBody(r io.ReadCloser) error {
	if r == nil {
		return fmt.Errorf("no json body provided")
	}

	return json.NewDecoder(r).Decode(d)
}

// ToJson serializes the deployment to json
func (d *Deployment) ToJson() []byte {
	data, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return data
}

// StateIs returns true when the internal state matches the check state
func (d *Deployment) StateIs(s string) bool {
	return d.state.Is(s)
}

// State returns true when the internal state of the deployment
func (d *Deployment) State() string {
	return d.state.Current()
}

// Configure the deployment and create any necessary configuration
func (d *Deployment) Configure() error {
	// callback executed after work is complete
	done := func() {
		// work has completed successfully
		go d.state.Event(EventConfigured)
	}

	// trigger the configure event
	return d.state.Event(EventConfigure, done)
}

func (d *Deployment) Deploy() error {
	// callback executed after work is complete
	done := func() {
		// work has completed successfully
		go d.state.Event(EventDeployed)
	}

	return d.state.Event(EventDeploy, done)
}
