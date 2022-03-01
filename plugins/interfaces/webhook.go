package interfaces

type WebhookMessage struct {
	Title            string
	Name             string
	Namespace        string
	State            string
	Outcome          string
	PrimaryTraffic   int
	CandidateTraffic int
	Error            string
}

type Webhook interface {
	Configurable

	// Send makes an outbound webhook call
	Send(message WebhookMessage) error
}
