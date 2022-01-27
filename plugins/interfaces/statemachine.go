package interfaces

import "time"

// StateHistory is a struct that defines the state at a point in time
type StateHistory struct {
	Time  time.Time
	State string
}

type StateMachine interface {
	// Configure triggers the EventConfigure state
	Configure() error

	// Deploy triggers the EventDeploy state
	Deploy() error

	// Destroy triggers the event Destroy state
	Destroy() error

	// CurrentState returns the current state
	CurrentState() string

	// StateHistory returns all the states for the statemachine
	StateHistory() []StateHistory
}
