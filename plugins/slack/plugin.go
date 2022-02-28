package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/DisgoOrg/log"
	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type slackClient interface {
	Send(url, payload string) error
}

type slackImpl struct{}

func (s *slackImpl) Send(url, content string) error {
	payload := slack.Payload{
		Text: content,
	}

	err := slack.Send(url, "", payload)
	if len(err) > 0 {
		return fmt.Errorf("unable to call Slack webhook: %s", err)
	}

	return nil
}

type Plugin struct {
	log    hclog.Logger
	config *PluginConfig
	client slackClient
}

type PluginConfig struct {
	URL      string `json:"url" validate:"required"`
	Template string `json:"template"`
}

func New(l hclog.Logger) (*Plugin, error) {
	return &Plugin{log: l}, nil
}

var ErrMissingURL = fmt.Errorf(`ID is a required field when configuring Discord webhooks, 
	you can obtain this value from the webhook URL in the Discord UI ('https://discord.com/api/webhooks/{id}/{token}')`)

func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}

	err := json.Unmarshal(data, p.config)
	if err != nil {
		return fmt.Errorf("unable to decode Webhook config: %s", err)
	}

	validate := validator.New()
	err = validate.Struct(p.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.URL":
				errorMessage += ErrMissingURL.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	p.client = &slackImpl{}

	return nil
}

func (p *Plugin) Send(message interfaces.WebhookMessage) error {
	log.Debug("Sending message to Slack", "url", p.config.URL, "message", message)

	templateContent := defaultContent
	if p.config.Template != "" {
		templateContent = p.config.Template
	}

	tmpl, err := template.New("slack").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("unable to process message template for Webhook plugin: %s", err)
	}

	out := bytes.NewBufferString("")
	err = tmpl.Execute(out, message)
	if err != nil {
		return fmt.Errorf("unable to execute template for Webhook plugin: %s", err)
	}

	return p.client.Send(p.config.URL, out.String())
}

var defaultContent = `
Consul Release Controller state has changed to "{{ .State }}" for
the release "{{ .Name }}" in the namespace "{{ .Namespace }}".

{{ if ne .Error "" }}
An error occurred when processing: {{ .Error }}
{{ else }}
The outcome was "{{ .Outcome }}"
{{ end }}
`
