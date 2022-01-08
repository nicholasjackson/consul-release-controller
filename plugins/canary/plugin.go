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
	config         *PluginConfig
	monitoring     interfaces.Monitor
	currentTraffic int
}

type PluginConfig struct {
	Interval             string `hcl:"interval,optional" json:"interval,omitempty"`
	InitialTraffic       int    `hcl:"initial_traffic,optional" json:"initial_traffic,omitempty"`
	TrafficStep          int    `hcl:"traffic_step,optional" json:"traffic_step,omitempty"`
	MaxTraffic           int    `hcl:"max_traffic,optional" json:"max_traffic,omitempty"`
	ErrorThreshold       int    `hcl:"error_threshold,optional" json:"error_threshold,omitempty"`
	DeleteCanaryOnFailed bool   `hcl:"delete_canary_on_failed,optional" json:"delete_canary_on_failed,omitempty"`
	ManualPromotion      bool   `hcl:"manual_promotion,optional" json:"manual_promotion"`
}

func New(l hclog.Logger, m interfaces.Monitor) (*Plugin, error) {
	return &Plugin{log: l, monitoring: m, currentTraffic: -1}, nil
}

// Configure the plugin with the given json
func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}

	return json.Unmarshal(data, p.config)
}

// Execute the strategy
func (p *Plugin) Execute(ctx context.Context) (interfaces.StrategyStatus, int, error) {
	p.log.Debug("initializing strategy", "type", "canary", "traffic", p.currentTraffic)

	// sleep for duration before checking
	d, err := time.ParseDuration(p.config.Interval)
	if err != nil {
		return interfaces.StrategyStatusFail, 0, fmt.Errorf("unable to parse interval: %s", err)
	}

	failCount := 0
	for {
		time.Sleep(d)
		p.log.Debug("executing strategy", "type", "canary")

		err := p.monitoring.Check(ctx)
		if err != nil {
			p.log.Debug("check failed", "type", "canary", "error", err)
			failCount++

			if failCount >= p.config.ErrorThreshold {
				return interfaces.StrategyStatusFail, 0, nil
			}

			continue
		}

		// no error returned
		if p.currentTraffic == -1 {
			p.currentTraffic = p.config.InitialTraffic
		} else {
			p.currentTraffic += p.config.TrafficStep
		}

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
