package interfaces

type Webhook interface {
	Configurable

	// Send makes an outbound webhook call
	Send(title, content string) error
}
