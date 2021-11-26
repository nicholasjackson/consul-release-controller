package metrics

type Metrics interface {
	// ServiceStarting is a counter that tracks the time the service has started
	ServiceStarting()
	HandleRequest(handler string, args map[string]string) func(status int)
}
