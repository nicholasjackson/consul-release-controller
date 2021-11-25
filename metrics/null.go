package metrics

// Null is a noop metrics sink
type Null struct {
}

func (n *Null) ServiceStarting() {}
