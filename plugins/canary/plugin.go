package canary

import "time"

type PluginConfig struct {
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
