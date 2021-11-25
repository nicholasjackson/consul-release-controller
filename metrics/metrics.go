package metrics

type Metrics interface {
	// ServiceStarting is a counter that tracks the time the service has started
	ServiceStarting()
}
