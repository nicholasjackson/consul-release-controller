package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/looplab/fsm"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	promMetrics "github.com/nicholasjackson/consul-canary-controller/metrics"
)

type Clients struct {
	Consul clients.Consul
}

type Deployment struct {
	Active    bool
	StartTime time.Time
	EndTime   time.Time

	ConsulService string              `hcl:"consul_service" json:"consul_service"`
	Kubernetes    *KubernetesWorkload `hcl:"kubernetes_workload" json:"kubernetes,omitempty"`
	Canary        *StrategyCanary     `hcl:"canary" json:"canary,omitempty"`

	state   *fsm.FSM
	log     hclog.Logger
	metrics promMetrics.Metrics
	clients *Clients
}

// NewDeployment creates a new deployment setting the state to inactive
func NewDeployment(log hclog.Logger, metrics promMetrics.Metrics, clients *Clients) *Deployment {
	d := &Deployment{}
	fsm := newFSM(d)

	d.state = fsm
	d.log = log
	d.metrics = metrics
	d.clients = clients

	return d
}

type KubernetesWorkload struct {
	Deployment string `hcl: "deployment" json:"deployment"`
}

type StrategyCanary struct {
	Interval             time.Duration `hcl:"interval,optional" json:"interval,omitempty"`
	InitialTraffic       int           `hcl:"initial_traffic,optional" json:"initial_traffic,omitempty"`
	TrafficStep          int           `hcl:"traffic_step,optional" json:"traffic_step,omitempty"`
	MaxTraffic           int           `hcl:"max_traffic,optional" json:"max_traffic,omitempty"`
	ErrorThreshold       int           `hcl:"error_threshold,optional" json:"error_threshold,omitempty"`
	DeleteCanaryOnFailed bool          `hcl:"delete_canary_on_failed,optional" json:"delete_canary_on_failed,omitempty"`
	ManualPromotion      bool          `hcl:"manual_promotion,optional" json:"manual_promotion"`

	Checks []*Check `hcl: "check" json:"checks,omitempty"`
}

type Check struct {
	Metric string `hcl:"metric" json:"metric"`
	Min    int    `hcl:"min,optional" json:"min,omitempty"`
	Max    int    `hcl:"min,optional" json:"max,omitempty"`
}

type Metric struct {
	Type  string `hcl:"type" json:"type"`
	Query string `hcl:"query" json:"query"`
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

// Initialize the deployment and create any necessary configuration
func (d *Deployment) Initialize() error {
	return d.state.Event(EventInitialize)
}

// initialize is an internal function triggered by the initialize event
func (d *Deployment) initialize(e *fsm.Event) {
	d.log.Debug("initializing deployment", "service", d.ConsulService)

	// create the service defaults for the primary and the canary
	err := d.clients.Consul.CreateServiceDefaults(fmt.Sprintf("cc-%s-primary", d.ConsulService))
	if err != nil {
		e.Cancel(err)
		return
	}

	err = d.clients.Consul.CreateServiceDefaults(fmt.Sprintf("cc-%s-canary", d.ConsulService))
	if err != nil {
		e.Cancel(err)
		return
	}

	// create the service resolver
	err = d.clients.Consul.CreateServiceResolver(fmt.Sprintf("cc-%s", d.ConsulService))
	if err != nil {
		e.Cancel(err)
		return
	}

	// create the service router
	err = d.clients.Consul.CreateServiceRouter(fmt.Sprintf("cc-%s", d.ConsulService))
	if err != nil {
		e.Cancel(err)
		return
	}

	// create the service spiltter set to 100% primary
	err = d.clients.Consul.CreateServiceSplitter(fmt.Sprintf("cc-%s", d.ConsulService), 100, 0)
	if err != nil {
		e.Cancel(err)
		return
	}

}
