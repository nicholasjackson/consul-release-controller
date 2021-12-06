package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
)

type Release struct {
	Name string `json:"name"`

	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`

	CurrentState string `json:"current_state"`

	Releaser *PluginConfig `json:"releaser"`
	Runtime  *PluginConfig `json:"runtime"`

	state *fsm.FSM

	releaserPlugin plugins.Releaser
	runtimePlugin  plugins.Runtime
}

type PluginConfig struct {
	Name   string          `json:"plugin_name"`
	Config json.RawMessage `json:"config"`
}

// Build creates a new deployment setting the state to inactive
// unless current state is set, this indicates that the release
// has been de-serialzed
func (d *Release) Build(pluginProvider plugins.Provider) error {
	// configure the setup plugin
	sp, err := pluginProvider.CreateReleaser(d.Releaser.Name)
	if err != nil {
		return err
	}

	// configure the releaser plugin
	sp.Configure(d.Releaser.Config)
	d.releaserPlugin = sp

	// configure the runtime plugin
	rp, err := pluginProvider.CreateRuntime(d.Runtime.Name)
	if err != nil {
		return err
	}

	// configure the runtime plugin
	rp.Configure(d.Runtime.Config)
	d.runtimePlugin = rp

	fsm := newFSM(d, sp, rp)
	d.state = fsm

	// if we are rehydrating this we probably have an existing state
	if d.CurrentState != "" {
		d.state.SetState(d.CurrentState)
	}

	return err
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
	// serialize the current state
	d.CurrentState = d.State()

	data, err := json.Marshal(d)
	if err != nil {
		panic(err)
	}

	return data
}

// RuntimePlugin returns the runtime plugin for this release
func (d *Release) RuntimePlugin() plugins.Runtime {
	return d.runtimePlugin
}

func (d *Release) SetState(s string) {
	d.state.SetState(s)
}

// Save release to the datastore
func (d *Release) Save(state string) {
	d.CurrentState = state
}

// StateIs returns true when the internal state matches the check state
func (d *Release) StateIs(s string) bool {
	if d.state == nil {
		return false
	}

	return d.state.Is(s)
}

// State returns true when the internal state of the deployment
func (d *Release) State() string {
	if d.state == nil {
		return ""
	}

	return d.state.Current()
}

// Configure the deployment and create any necessary configuration
func (d *Release) Configure() error {
	// callback executed after work is complete
	done := func(e *fsm.Event) {
		// work has completed successfully
		go d.state.Event(EventConfigured)
	}

	// trigger the configure event
	return d.state.Event(EventConfigure, done)
}

func (d *Release) Deploy() error {
	// callback executed after work is complete
	done := func(e *fsm.Event) {
		// work has completed successfully
		go d.state.Event(EventDeployed)
	}

	return d.state.Event(EventDeploy, done)
}
