package interfaces

import (
	"encoding/json"

	"golang.org/x/net/context"
)

type StrategyStatus string

const (
	StrategyStatusSuccess  = "strategy_status_success"
	StrategyStatusFail     = "strategy_status_fail"
	StrategyStatusComplete = "strategy_status_complete"
)

// Strategy defines the interface for a roll out strategy like a Canary or a Blue/Green
type Strategy interface {
	// Configure the plugin with the given json
	Configure(name, namespace string, config json.RawMessage) error

	// Execute the strategy and return the StrategyStatus on a successfull check
	// when StrategyStatusSuccess is returned the new traffic amount to be sent to the service is returned
	Execute(ctx context.Context) (status StrategyStatus, traffic int, err error)
}
