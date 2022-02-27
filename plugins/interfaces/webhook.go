package interfaces

type WebhookMessage struct {
	Title     string
	Name      string
	Namespace string
	State     string
	Outcome   string
	Error     error
}

type Webhook interface {
	Configurable

	// Send makes an outbound webhook call
	Send(message WebhookMessage) error
}
