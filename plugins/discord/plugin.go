package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgo/rest"
	"github.com/DisgoOrg/disgo/webhook"
	"github.com/DisgoOrg/log"
	"github.com/DisgoOrg/snowflake"
	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type discordClient interface {
	CreateEmbeds(embeds []discord.Embed, opts ...rest.RequestOpt) (*webhook.Message, error)
}

type Plugin struct {
	log    hclog.Logger
	config *PluginConfig
	client discordClient
}

type PluginConfig struct {
	ID       string `json:"id" validate:"required"`
	Token    string `json:"token" validate:"required"`
	Template string `json:"template"`
}

func New(l hclog.Logger) (*Plugin, error) {
	return &Plugin{log: l}, nil
}

var ErrMissingID = fmt.Errorf(`ID is a required field when configuring Discord webhooks, 
	you can obtain this value from the webhook URL in the Discord UI ('https://discord.com/api/webhooks/{id}/{token}')`)

var ErrMissingToken = fmt.Errorf(`Token is a required field when configuring Discord webhooks, 
	you can obtain this value from the webhook URL in the Discord UI ('https://discord.com/api/webhooks/{id}/{token}')`)

func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}

	err := json.Unmarshal(data, p.config)
	if err != nil {
		return fmt.Errorf("unable to decode Monitoring config: %s", err)
	}

	validate := validator.New()
	err = validate.Struct(p.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.ID":
				errorMessage += ErrMissingID.Error() + "\n"
			case "PluginConfig.Token":
				errorMessage += ErrMissingToken.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	p.client = webhook.NewClient(snowflake.Snowflake(p.config.ID), p.config.Token)

	return nil
}

func (p *Plugin) Send(message interfaces.WebhookMessage) error {
	log.Debug("Sending message to Discord", "id", p.config.ID, "message", message)

	templateContent := defaultContent
	if p.config.Template != "" {
		templateContent = p.config.Template
	}

	tmpl, err := template.New("discord").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("unable to process message template for Webhook plugin: %s", err)
	}

	out := bytes.NewBufferString("")
	err = tmpl.Execute(out, message)
	if err != nil {
		return fmt.Errorf("unable to execute template for Webhook plugin: %s", err)
	}

	_, err = p.client.CreateEmbeds([]discord.Embed{
		discord.NewEmbedBuilder().
			SetTitle(message.Title).
			SetDescription(out.String()).
			Build(),
	})

	if err != nil {
		return fmt.Errorf("unable to make Webhook call to Discord: %s", err)
	}

	return nil
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
