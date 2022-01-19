package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/armon/go-metrics/prometheus"

	"github.com/armon/go-metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Sink struct {
	addr   string
	port   int
	path   string
	m      *metrics.Metrics
	prom   *prometheus.PrometheusSink
	server *http.Server
}

func New(addr string, port int, path string) (*Sink, error) {
	promSink, err := prometheus.NewPrometheusSink()
	if err != nil {
		return nil, err
	}

	m, err := metrics.New(metrics.DefaultConfig("consul_canary_controller"), promSink)
	if err != nil {
		return nil, err
	}

	m.EnableRuntimeMetrics = true

	return &Sink{addr: addr, port: port, path: path, m: m, prom: promSink}, nil
}

// StartServer exposes the metrics
func (m *Sink) StartServer() error {
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

func (m *Sink) StopServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return m.server.Shutdown(ctx)
}

func (m *Sink) ServiceStarting() {
	m.m.IncrCounter([]string{"starting"}, 1)
}

func (m *Sink) HandleRequest(handler string, args map[string]string) func(status int) {
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
