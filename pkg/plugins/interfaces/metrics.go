package interfaces

// Metrics defines an interface that metrics reporting plugins must implement
type Metrics interface {
	// ServiceStarting is a counter that tracks the time the service has started
	ServiceStarting()
	// HandleRequest records the duration of the HTTP API handlers
	HandleRequest(handler string, args map[string]string) func(status int)
	// StateChanged records the duration of statemachine changes
	StateChanged(release, state string, args map[string]string) func(status int)
}
