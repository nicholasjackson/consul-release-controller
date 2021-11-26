package metrics

// Null is a noop metrics sink
type Null struct {
}

func (n *Null) ServiceStarting() {}
func (n *Null) HandleRequest(handler string, args map[string]string) func(status int) {
	return func(status int) {}
}
