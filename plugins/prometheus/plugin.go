package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/prometheus/common/model"

	"github.com/hashicorp/go-hclog"
)

type Plugin struct {
	log    hclog.Logger
	config *PluginConfig
	client clients.Prometheus
}

type PluginConfig struct {
	// Address of the prometheus server
	Address string  `json:"address"`
	Queries []Query `json:"queries"`
}

// Query config
type Query struct {
	// Name of the query
	Name string `json:"name"`

	// Preset is an optional default metric query
	Preset string `json:"preset"`

	// Minimum value for success, optional when Max specified
	Min *int `json:"min,omitempty"` // default 0

	// Maximum value for success, optional when Min specified
	Max *int `json:"max,omitempty"` // default 0
}

func New(l hclog.Logger) (*Plugin, error) {
	c, _ := clients.NewPrometheus()
	return &Plugin{log: l, client: c}, nil
}

func (s *Plugin) Configure(data json.RawMessage) error {
	s.config = &PluginConfig{}

	err := json.Unmarshal(data, s.config)
	if err != nil {
		return fmt.Errorf("unable to decode Monitoring config: %s", err)
	}

	return nil
}

// Check executes queries to the Prometheus server and returns an error if any of the queries
// are not within the defined min and max thresholds
func (s *Plugin) Check(ctx context.Context, name, namespace string, interval time.Duration) error {
	querySQL := []string{}

	// first check that the given queries have valid presets
	for _, q := range s.config.Queries {
		switch q.Preset {
		case "kubernetes-envoy-request-success":
			querySQL = append(querySQL, KubernetesEnvoyRequestSuccess)
		case "kubernetes-envoy-request-duration":
			querySQL = append(querySQL, KubernetesEnvoyRequestDuration)
		default:
			return fmt.Errorf("preset query %s, does not exist", q.Preset)
		}
	}

	// execute the queries
	for i, q := range querySQL {
		query := s.config.Queries[i]

		// add the interpolation for the queries
		tmpl, err := template.New("query").Parse(q)
		if err != nil {
			return err
		}

		context := struct {
			Name      string
			Namespace string
			Interval  string
		}{
			name,
			namespace,
			interval.String(),
		}

		out := bytes.NewBufferString("")
		err = tmpl.Execute(out, context)
		if err != nil {
			return err
		}

		s.log.Debug("querying prometheus", "address", s.config.Address, "name", query.Name, "query", out)

		val, warn, err := s.client.Query(ctx, s.config.Address, out.String(), time.Now())
		if err != nil {
			s.log.Error("unable to query prometheus", "error", err)

			return err
		}

		s.log.Debug("query value returned", "name", query.Name, "preset", query.Preset, "value", val, "warnings", warn)

		if v, ok := val.(*model.Scalar); ok {
			checkFail := false
			if query.Min != nil && int(v.Value) < *query.Min {
				s.log.Debug("query value less than min", "name", query.Name, "preset", query.Preset, "value", v.Value)
				checkFail = true
			}

			if query.Max != nil && int(v.Value) > *query.Max {
				s.log.Debug("query value greater than max", "name", query.Name, "preset", query.Preset, "value", v.Value)
				checkFail = true
			}

			if checkFail {
				return fmt.Errorf("check failed for query %s using preset %s, got value %d", query.Name, query.Preset, int(v.Value))
			}
		}
	}

	return nil
}
