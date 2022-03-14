package canary

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type Plugin struct {
	log            hclog.Logger
	config         *PluginConfig
	monitoring     interfaces.Monitor
	currentTraffic int
}

type PluginConfig struct {
	// InitialDelay before configuring the first traffic split
	InitialDelay string `hcl:"initial_delay,optional" json:"initial_delay,omitempty" validate:"duration"`
	// Interval between checks
	Interval string `hcl:"interval,optional" json:"interval,omitempty" validate:"duration"`
	// InitialTraffic percentage to send to the canary
	InitialTraffic int `hcl:"initial_traffic,optional" json:"initial_traffic,omitempty" validate:"gte=0,lte=100"`
	// TrafficStep is the percentage of traffic to increase with every step
	TrafficStep int `hcl:"traffic_step,optional" json:"traffic_step,omitempty" validate:"gte=1,lte=100,required"`
	// MaxTraffic to send to the canary before promoting to primary
	MaxTraffic int `hcl:"max_traffic,optional" json:"max_traffic,omitempty" validate:"gte=1,lte=100,required"`
	// ErrorThreshold is the number of consecutive failed checks before rolling back traffic
	ErrorThreshold int `hcl:"error_threshold,optional" json:"error_threshold,omitempty" validate:"required,gte=0"`
	// DeleteCanaryOnFailed determines if the canary deployment is deleted on a failed check
	DeleteCanaryOnFailed bool `hcl:"delete_canary_on_failed,optional" json:"delete_canary_on_failed,omitempty"`
	// ManualPromotion requires manual intervention before the canary is promoted to primary
	ManualPromotion bool `hcl:"manual_promotion,optional" json:"manual_promotion"`
}

var ErrInvalidInitialDelay = fmt.Errorf("InitialDelay is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")
var ErrInvalidInterval = fmt.Errorf("Interval is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")
var ErrInvalidInitialTraffic = fmt.Errorf("InitialTraffic must contain a value between 0 and 100")
var ErrTrafficStep = fmt.Errorf("TrafficStep must contain a value between 1 and 100")
var ErrMaxTraffic = fmt.Errorf("MaxTraffic must contain a value between 1 and 100")
var ErrThreshold = fmt.Errorf("ErrorThreshold must contain a value greater than 0")

func New(l hclog.Logger, m interfaces.Monitor) (*Plugin, error) {
	return &Plugin{log: l, monitoring: m, currentTraffic: -1}, nil
}

// Configure the plugin with the given json
// returns an error when validation fails for the config
func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}

	err := json.Unmarshal(data, p.config)
	if err != nil {
		return err
	}

	// if no initial delay use the interval
	if p.config.InitialDelay == "" {
		p.config.InitialDelay = p.config.Interval
	}

	// validate the plugin config
	validate := validator.New()
	validate.RegisterValidation("duration", interfaces.ValidateDuration)
	err = validate.Struct(p.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.InitialDelay":
				errorMessage += ErrInvalidInitialDelay.Error() + "\n"
			case "PluginConfig.Interval":
				errorMessage += ErrInvalidInterval.Error() + "\n"
			case "PluginConfig.InitialTraffic":
				errorMessage += ErrInvalidInitialTraffic.Error() + "\n"
			case "PluginConfig.TrafficStep":
				errorMessage += ErrTrafficStep.Error() + "\n"
			case "PluginConfig.MaxTraffic":
				errorMessage += ErrMaxTraffic.Error() + "\n"
			case "PluginConfig.ErrorThreshold":
				errorMessage += ErrThreshold.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	return nil
}

// Execute the strategy
// interfaces.StrategyStatusSuccess and the percentage of traffic to set to the canditate returned on success of the checks
// interfaces.StrategyStatusFail and the percentage of traffic to set to the canditate returned on failure of the checks
// interfaces.StrategyStatusFail and an error is returned on an internal error
func (p *Plugin) Execute(ctx context.Context) (interfaces.StrategyStatus, int, error) {
	p.log.Info("Executing strategy", "type", "canary", "traffic", p.currentTraffic)

	// if this is the first run set the initial traffic and return
	if p.currentTraffic == -1 {
		p.currentTraffic = p.config.InitialTraffic

		if p.config.InitialTraffic == 0 {
			p.currentTraffic = p.config.TrafficStep
		}

		d, err := time.ParseDuration(p.config.InitialDelay)
		if err != nil {
			return interfaces.StrategyStatusFail, 0, fmt.Errorf("unable to parse initial delay: %s", err)
		}

		p.log.Debug("Waiting for initial grace before starting rollout", "type", "canary", "delay", d.Seconds())
		time.Sleep(d)

		p.log.Debug("Strategy setup", "type", "canary", "traffic", p.currentTraffic)
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

		p.log.Debug("Checking metrics", "type", "canary")

		err := p.monitoring.Check(queryCtx, d)
		if err != nil {
			p.log.Debug("Check failed", "type", "canary", "error", err)
			failCount++

			if failCount >= p.config.ErrorThreshold {
				// reset the state
				p.currentTraffic = -1
				return interfaces.StrategyStatusFail, 0, nil
			}

			continue
		}

		p.currentTraffic += p.config.TrafficStep

		if p.currentTraffic >= p.config.MaxTraffic {
			// strategy is complete
			p.log.Debug("Strategy complete", "type", "canary", "traffic", p.currentTraffic)

			// reset the state
			p.currentTraffic = -1
			return interfaces.StrategyStatusComplete, 100, nil
		}

		// step has been successful
		p.log.Debug("Strategy success", "type", "canary", "traffic", p.currentTraffic)
		return interfaces.StrategyStatusSuccess, p.currentTraffic, nil
	}
}

func (p *Plugin) GetPrimaryTraffic() int {
	return 100 - p.currentTraffic
}

func (p *Plugin) GetCandidateTraffic() int {
	return p.currentTraffic
}
