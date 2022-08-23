package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/armon/go-metrics/prometheus"

	"github.com/armon/go-metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var globalMetrics *metrics.Metrics

// Metrics defines a metrics server that can collect and report application metrics
type Metrics struct {
	addr   string
	port   int
	path   string
	server *http.Server
}

// NewMetrics creates a new metrics server
func NewMetrics(addr string, port int, path string) (*Metrics, error) {
	if globalMetrics == nil {
		promSink, err := prometheus.NewPrometheusSink()
		if err != nil {
			return nil, fmt.Errorf("unable to create Prometheus metrics sink: %s", err)
		}

		config := metrics.DefaultConfig("consul_release_controller")
		config.EnableHostname = false

		mets, err := metrics.New(config, promSink)
		if err != nil {
			return nil, fmt.Errorf("unable to intiialzie new metrics server: %s", err)
		}

		mets.EnableRuntimeMetrics = true

		globalMetrics = mets
	}

	return &Metrics{addr: addr, port: port, path: path}, nil
}

// StartServer exposes the metrics
func (m *Metrics) StartServer() error {
	mux := &http.ServeMux{}
	mux.Handle(m.path, promhttp.Handler())

	errChan := make(chan error)
	timeout := time.After(1500 * time.Millisecond)

	// start the server in the background but wait to
	// check that it can bind correctly
	// if not return an error
	m.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", m.addr, m.port),
		Handler: mux,
	}
	go func() {
		errChan <- m.server.ListenAndServe()
	}()

	select {
	case <-timeout:
		return nil
	case e := <-errChan:
		return e
	}

}

// StopServer stops the metrics server
func (m *Metrics) StopServer() error {
	if m.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := m.server.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// ServiceStarting is the metrics handler for when the release controller service has started
func (m *Metrics) ServiceStarting() {
	globalMetrics.IncrCounter([]string{"starting"}, 1)
}

// HandleRequest is a generic metrics handler for HTTP requests
func (m *Metrics) HandleRequest(handler string, args map[string]string) func(status int) {
	st := time.Now()

	return func(status int) {
		labs := []metrics.Label{}
		for k, v := range args {
			labs = append(labs, metrics.Label{Name: k, Value: v})
		}

		// add the response code
		labs = append(labs, metrics.Label{Name: "response_code", Value: fmt.Sprintf("%d", status)})

		globalMetrics.MeasureSinceWithLabels([]string{handler}, st, labs)
	}
}

// StateChanged is a metrics handler to update the state of a release
func (m *Metrics) StateChanged(release, state string, args map[string]string) func(status int) {
	st := time.Now()

	labs := []metrics.Label{metrics.Label{Name: "release", Value: release}, metrics.Label{Name: "state", Value: state}}
	for k, v := range args {
		labs = append(labs, metrics.Label{Name: k, Value: v})
	}

	globalMetrics.SetGaugeWithLabels([]string{"state_changed_start_seconds"}, float32(time.Now().UnixNano()), labs)

	return func(status int) {
		// add the response code
		labs = append(labs, metrics.Label{Name: "response_code", Value: fmt.Sprintf("%d", status)})

		globalMetrics.MeasureSinceWithLabels([]string{"state_change_duration"}, st, labs)
	}
}
