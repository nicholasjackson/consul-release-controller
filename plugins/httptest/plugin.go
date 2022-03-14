package httptest

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
)

type Plugin struct {
	log        hclog.Logger
	config     *PluginConfig
	monitoring interfaces.Monitor
}

type PluginConfig struct {
	// Path of the service endpoint to call
	Path string `hcl:"path" json:"path" validate:"required,uri"`

	// Method is the HTTP method for the test GET,POST,HEAD,etc
	Method string `hcl:"method" json:"method,omitempty" validate:"required,oneof=GET POST PUT DELETE HEAD OPTIONS TRACE PATCH"`

	// Payload is sent along with POST or PUT requests
	Payload string `hcl:"payload,optional" json:"payload,omitempty"`

	// Interval between checks
	Interval string `hcl:"interval" json:"interval" validate:"required,duration"`

	// Duration to run the tests for
	Duration string `hcl:"duration" json:"duration" validate:"required,duration"`
}

var ErrInvalidPath = fmt.Errorf("Path is not a valid HTTP path")
var ErrInvalidMethod = fmt.Errorf("Path is not a valid HTTP method, please specify one of GET,POST,PUT,DELETE,HEAD,OPTIONS,TRACE,PATCH")
var ErrInvalidInterval = fmt.Errorf("Interval is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")
var ErrInvalidDuration = fmt.Errorf("Duration is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")

func New(l hclog.Logger, m interfaces.Monitor) (*Plugin, error) {
	return &Plugin{log: l, monitoring: m}, nil
}

// Configure the plugin with the given json
// returns an error when validation fails for the config
func (p *Plugin) Configure(data json.RawMessage) error {
	p.config = &PluginConfig{}

	err := json.Unmarshal(data, p.config)
	if err != nil {
		return err
	}

	// validate the plugin config
	validate := validator.New()
	validate.RegisterValidation("duration", interfaces.ValidateDuration)
	err = validate.Struct(p.config)

	if err != nil {
		errorMessage := ""
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Namespace() {
			case "PluginConfig.Path":
				errorMessage += ErrInvalidPath.Error() + "\n"
			case "PluginConfig.Method":
				errorMessage += ErrInvalidMethod.Error() + "\n"
			case "PluginConfig.Interval":
				errorMessage += ErrInvalidInterval.Error() + "\n"
			case "PluginConfig.Duration":
				errorMessage += ErrInvalidDuration.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	return nil
}
