package mocks

import "github.com/stretchr/testify/mock"

type MetricsMock struct {
	mock.Mock
}

func (m *MetricsMock) ServiceStarting() {
	m.Called()
}

// HandleRequest records the duration of the HTTP API handlers
func (m *MetricsMock) HandleRequest(handler string, args map[string]string) func(status int) {
	returnArgs := m.Called(handler, args)

	return returnArgs.Get(0).(func(status int))
}

// StateChanged records the duration of statemachine changes
func (m *MetricsMock) StateChanged(release, state string, args map[string]string) func(status int) {
	returnArgs := m.Called(release, state, args)

	return returnArgs.Get(0).(func(status int))
}
