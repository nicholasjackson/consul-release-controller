package interfaces

import (
	"golang.org/x/net/context"
)

type StrategyStatus string

const (
	StrategyStatusSuccess  = "strategy_status_progressing"
	StrategyStatusFailing  = "strategy_status_failing"
	StrategyStatusFailed   = "strategy_status_failed"
	StrategyStatusComplete = "strategy_status_complete"
)

// Strategy defines the interface for a roll out strategy like a Canary or a Blue/Green
type Strategy interface {
	Configurable

	// Execute the strategy and return the StrategyStatus on a successful check
	// when StrategyStatusSuccess is returned the new traffic amount to be sent to the service is returned
	Execute(ctx context.Context, candidateName string) (status StrategyStatus, traffic int, err error)

	// GetPrimaryTraffic returns the percentage of traffic distributed to the primary instance
	GetPrimaryTraffic() int
	// GetCandidateTraffic returns the percentage of traffic distributed to the candidate instance
	GetCandidateTraffic() int
}
