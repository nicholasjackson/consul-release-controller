package httptest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/config"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
)

type Plugin struct {
	log        hclog.Logger
	store      interfaces.PluginStateStore
	config     *PluginConfig
	monitoring interfaces.Monitor

	name      string
	namespace string
	runtime   string
}

type PluginConfig struct {
	// Path of the service endpoint to call
	Path string `hcl:"path" json:"path" validate:"required,uri"`

	// Method is the HTTP method for the test GET,POST,HEAD,etc
	Method string `hcl:"method" json:"method,omitempty" validate:"required,oneof=GET POST PUT DELETE HEAD OPTIONS TRACE PATCH"`

	// Payload is sent along with POST or PUT requests
	Payload string `hcl:"payload,optional" json:"payload,omitempty"`

	// RequiredTestPasses is the number of continuous successful test checks that must be attained before the
	// PostDeploymentTest is returned as healthy. E.g. if this value is set to 5 then there must be 5 tests
	// tests that result in a successfull outcome without error. A single failure resets the pass count to 0, i.e.
	// 6 tests that are a mixture of 5 passes but contain 1 failure (p p p f p p) would not result in an overall pass
	// until three more positive results have been obtained as the pass count is reset on the first failure.
	RequiredTestPasses int `hcl:"required_test_passes" json:"required_test_passes" validate:"required,gte=1"`

	// Interval between checks
	Interval string `hcl:"interval" json:"interval" validate:"required,duration"`

	// Timeout specifies the maximum duration the tests will run for
	Timeout string `hcl:"timeout" json:"timeout" validate:"required,duration"`
}

var ErrInvalidPath = fmt.Errorf("Path is not a valid HTTP path")
var ErrInvalidMethod = fmt.Errorf("Method is not a valid HTTP method, please specify one of GET,POST,PUT,DELETE,HEAD,OPTIONS,TRACE,PATCH")
var ErrInvalidInterval = fmt.Errorf("Interval is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")
var ErrInvalidTimeout = fmt.Errorf("Timeout is not a valid duration, please specify using Go duration format e.g (30s, 30ms, 60m)")
var ErrInvalidTestPasses = fmt.Errorf("RequiredTestPasses is not valid, please specify a value greater than 0")

func New(name, namespace, runtime string, m interfaces.Monitor) (*Plugin, error) {
	// if there is no namespaces set, then use the convention for default to ensure the upstream routing works
	if namespace == "" {
		namespace = "default"
	}

	return &Plugin{monitoring: m, name: name, namespace: namespace, runtime: runtime}, nil
}

// Configure the plugin with the given json
// returns an error when validation fails for the config
func (p *Plugin) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	p.log = log
	p.store = store
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
			case "PluginConfig.Timeout":
				errorMessage += ErrInvalidTimeout.Error() + "\n"
			case "PluginConfig.RequiredTestPasses":
				errorMessage += ErrInvalidTestPasses.Error() + "\n"
			}
		}

		return fmt.Errorf(errorMessage)
	}

	return nil
}

func (p *Plugin) Execute(ctx context.Context, candidateName string) error {
	timeoutDuration, err := time.ParseDuration(p.config.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse timeout as duration: %s", err)
	}

	interval, err := time.ParseDuration(p.config.Interval)
	if err != nil {
		return fmt.Errorf("unable to parse interval as duration: %s", err)
	}
	successCount := 0
	timeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	for {
		// Make a call to the external service to an instance of Envoy proxy that exposes the different services using HOST header on the same port
		url := fmt.Sprintf("%s%s", config.ConsulServiceUpstreams(), p.config.Path)
		host := fmt.Sprintf("%s.%s", p.name, p.namespace)

		p.log.Debug("Executing request to upstream", "url", url, "upstream", host)

		httpreq, err := http.NewRequest(p.config.Method, url, bytes.NewBufferString(p.config.Payload))
		if err != nil {
			p.log.Error("Unable to create HTTP request", "error", err)
			return err
		}

		// The envoy proxy that is providing access to the candidate service has been configured to use HOST header to
		// differentiate between the services. The convention is service.namespace
		httpreq.Host = host

		resp, err := http.DefaultClient.Do(httpreq)
		if err != nil {
			return err
		}

		p.log.Debug("Response from upstream",
			"url", url,
			"upstream", host,
			"status_code", resp.StatusCode,
		)

		// We are ignoring the status code as we are using the Monitoring checks as a measure of success
		res, _ := p.monitoring.Check(ctx, candidateName, 30*time.Second)
		if res == interfaces.CheckSuccess {
			successCount++
		} else {
			// on failure reset the success count as passes must be continuous
			successCount = 0
		}

		switch {
		case timeout.Err() != nil:
			p.log.Error("Post deployment test failed, test timeout", "successCount", successCount)
			return fmt.Errorf("post deployment test failed, timeout waiting for successful tests")
		case successCount >= p.config.RequiredTestPasses:
			return nil
		}

		time.Sleep(interval)
	}
}
