package models

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Deployment struct {
	ConsulService string              `hcl: "consul_service" json:"consul_service"`
	Kubernetes    *KubernetesWorkload `hcl: "kubernetes_workload" json:"kubernetes,omitempty"`
	Canary        *StrategyCanary     `hcl: "canary" json:"canary,omitempty"`
}

type KubernetesWorkload struct {
	Deployment string `hcl: "deployment" json:"deployment"`
}

type StrategyCanary struct {
	Interval             time.Duration `hcl: "interval,optional" json:"interval,omitempty"`
	InitalTraffic        int           `hcl: "initial_traffic,optional" json:"initial_traffic,omitempty"`
	TrafficStep          int           `hcl: "traffic_step,optional" json:"traffic_step,omitempty"`
	MaxTraffic           int           `hcl: "max_traffic,optional" json:"max_traffic,omitempty"`
	ErrorThreshold       int           `hcl: "error_threshold,optional" json:"error_threshold,omitempty"`
	DeleteCanaryOnFailed bool          `hcl: "delete_canary_on_failed,optional" json:"delete_canary_on_failed,omitempty"`
	ManualPromotion      bool          `hcl: "manual_promotion,optional" json:"manual_promotion"`

	Checks []*Check `hcl: "check" json:"checks,omitempty"`
}

type Check struct {
	Metric string `hcl: "metric" json:"metric"`
	Min    int    `hcl: "min,optional" json:"min,omitempty"`
	Max    int    `hcl: "min,optional" json:"max,omitempty"`
}

type Metric struct {
	Type  string `hcl: "type" json:"type"`
	Query string `hcl: "query" json:"query"`
}

// FromJsonBody decodes the json body into the Deployment type
func (d *Deployment) FromJsonBody(r io.ReadCloser) error {
	if r == nil {
		return fmt.Errorf("no json body provided")
	}

	defer r.Close()
	return json.NewDecoder(r).Decode(d)
}
