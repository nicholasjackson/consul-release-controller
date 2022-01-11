package canary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
)

type Plugin struct {
	log            hclog.Logger
	name           string
	namespace      string
	config         *PluginConfig
	monitoring     interfaces.Monitor
	currentTraffic int
}

type PluginConfig struct {
	// InitialDelay before configuring the first traffic split
	InitialDelay string `hcl:"initial_delay,optional" json:"initial_delay,omitempty"`
	// Interval between checks
	Interval string `hcl:"interval,optional" json:"interval,omitempty"`
	// InitialTraffic percentage to send to the canary
	InitialTraffic int `hcl:"initial_traffic,optional" json:"initial_traffic,omitempty"`
	// TrafficStep is the percentage of traffic to increase with every step
	TrafficStep int `hcl:"traffic_step,optional" json:"traffic_step,omitempty"`
	// MaxTraffic to send to the canary before promoting to primary
	MaxTraffic int `hcl:"max_traffic,optional" json:"max_traffic,omitempty"`
	// ErrorThreshold is the number of consecutive failed checks before rolling back traffic
	ErrorThreshold int `hcl:"error_threshold,optional" json:"error_threshold,omitempty"`
	// DeleteCanaryOnFailed determines if the canary deployment is deleted on a failed check
	DeleteCanaryOnFailed bool `hcl:"delete_canary_on_failed,optional" json:"delete_canary_on_failed,omitempty"`
	// ManualPromotion requires manual intervention before the canary is promoted to primary
	ManualPromotion bool `hcl:"manual_promotion,optional" json:"manual_promotion"`
}

func New(l hclog.Logger, m interfaces.Monitor) (*Plugin, error) {
	return &Plugin{log: l, monitoring: m, currentTraffic: -1}, nil
}

// Configure the plugin with the given json
func (p *Plugin) Configure(name, namespace string, data json.RawMessage) error {
	p.config = &PluginConfig{}
	p.name = name
	p.namespace = namespace

	err := json.Unmarshal(data, p.config)

	// if no initial delay use the interval
	if p.config.InitialDelay == "" {
		p.config.InitialDelay = p.config.Interval
	}

	return err
}

// Execute the strategy
func (p *Plugin) Execute(ctx context.Context) (interfaces.StrategyStatus, int, error) {
	p.log.Info("executing strategy", "type", "canary", "traffic", p.currentTraffic)

	// if this is the first run set the initial traffic and return
	if p.currentTraffic == -1 {
		p.currentTraffic = p.config.InitialTraffic

		d, err := time.ParseDuration(p.config.InitialDelay)
		if err != nil {
			return interfaces.StrategyStatusFail, 0, fmt.Errorf("unable to parse initial delay: %s", err)
		}

		time.Sleep(d)

		p.log.Debug("strategy setup", "type", "canary", "traffic", p.currentTraffic)
		return interfaces.StrategyStatusSuccess, p.currentTraffic, nil
	}

	// sleep for duration before checking
	d, err := time.ParseDuration(p.config.Interval)
	if err != nil {
		return interfaces.StrategyStatusFail, 0, fmt.Errorf("unable to parse interval: %s", err)
	}

	failCount := 0
	for {
		time.Sleep(d)

		queryCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()

		p.log.Debug("checking metrics", "type", "canary")
		err := p.monitoring.Check(queryCtx, d)
		if err != nil {
			p.log.Debug("check failed", "type", "canary", "error", err)
			failCount++

			if failCount >= p.config.ErrorThreshold {
				return interfaces.StrategyStatusFail, 0, nil
			}

			continue
		}

		p.currentTraffic += p.config.TrafficStep

		if p.currentTraffic >= p.config.MaxTraffic {
			// strategy is complete
			p.log.Debug("strategy complete", "type", "canary", "traffic", p.currentTraffic)
			return interfaces.StrategyStatusComplete, 100, nil
		}

		// step has been successful
		p.log.Debug("strategy success", "type", "canary", "traffic", p.currentTraffic)
		return interfaces.StrategyStatusSuccess, p.currentTraffic, nil
	}
}
