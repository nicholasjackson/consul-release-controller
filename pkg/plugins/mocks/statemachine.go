package mocks

import (
	"github.com/stretchr/testify/mock"
)

type StateMachineMock struct {
	mock.Mock
}

// Configure triggers the EventConfigure state
func (sm *StateMachineMock) Configure() error {
	args := sm.Called()

	return args.Error(0)
}

// Deploy triggers the EventDeploy state
func (sm *StateMachineMock) Deploy() error {
	args := sm.Called()

	return args.Error(0)
}

// Destroy triggers the event Destroy state
func (sm *StateMachineMock) Destroy() error {
	args := sm.Called()

	return args.Error(0)
}

// Resume triggers the event Resume state
func (sm *StateMachineMock) Resume() error {
	args := sm.Called()

	return args.Error(0)
}

// CurrentState returns the current state
func (sm *StateMachineMock) CurrentState() string {
	args := sm.Called()

	return args.String(0)
}
