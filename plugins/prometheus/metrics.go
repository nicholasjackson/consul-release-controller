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

type Metrics struct {
	addr   string
	port   int
	path   string
	m      *metrics.Metrics
	prom   *prometheus.PrometheusSink
	server *http.Server
}

func NewMetrics(addr string, port int, path string) (*Metrics, error) {
	promSink, err := prometheus.NewPrometheusSink()
	if err != nil {
		return nil, err
	}

	config := metrics.DefaultConfig("consul_release_controller")
	config.EnableHostname = false

	m, err := metrics.New(config, promSink)
	if err != nil {
		return nil, err
	}

	m.EnableRuntimeMetrics = true

	return &Metrics{addr: addr, port: port, path: path, m: m, prom: promSink}, nil
}

// StartServer exposes the metrics
func (m *Metrics) StartServer() error {
	mux := &http.ServeMux{}
	mux.Handle(m.path, promhttp.Handler())

	err := make(chan error)
	timeout := time.After(500 * time.Millisecond)

	// start the server in the background but wait to
	// check that it can bind correctly
	// if not return an error
	m.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", m.addr, m.port),
		Handler: mux,
	}
	go func() {
		err <- m.server.ListenAndServe()
	}()

	select {
	case <-timeout:
		return nil
	case e := <-err:
		return e
	}

}

func (m *Metrics) StopServer() error {
	if m.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return m.server.Shutdown(ctx)
}

func (m *Metrics) ServiceStarting() {
	m.m.IncrCounter([]string{"starting"}, 1)
}

func (m *Metrics) HandleRequest(handler string, args map[string]string) func(status int) {
	st := time.Now()

	return func(status int) {
		labs := []metrics.Label{}
		for k, v := range args {
			labs = append(labs, metrics.Label{Name: k, Value: v})
		}

		// add the response code
		labs = append(labs, metrics.Label{Name: "response_code", Value: fmt.Sprintf("%d", status)})

		m.m.MeasureSinceWithLabels([]string{handler}, st, labs)
	}
}

func (m *Metrics) StateChanged(release, state string, args map[string]string) func(status int) {
	st := time.Now()

	labs := []metrics.Label{metrics.Label{Name: "release", Value: release}, metrics.Label{Name: "state", Value: state}}
	for k, v := range args {
		labs = append(labs, metrics.Label{Name: k, Value: v})
	}

	m.m.SetGaugeWithLabels([]string{"state_changed_start_seconds"}, float32(time.Now().UnixNano()), labs)

	return func(status int) {
		// add the response code
		labs = append(labs, metrics.Label{Name: "response_code", Value: fmt.Sprintf("%d", status)})

		m.m.MeasureSinceWithLabels([]string{"state_change_duration"}, st, labs)
	}
}
