package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/armon/go-metrics/prometheus"

	"github.com/armon/go-metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Sink struct {
	addr string
	port int
	path string
	m    *metrics.Metrics
	prom *prometheus.PrometheusSink
}

func New(addr string, port int, path string) (*Sink, error) {
	promSink, err := prometheus.NewPrometheusSink()
	if err != nil {
		return nil, err
	}

	m, err := metrics.NewGlobal(metrics.DefaultConfig("consul_canary_controller"), promSink)
	if err != nil {
		return nil, err
	}

	m.EnableRuntimeMetrics = true

	return &Sink{addr, port, path, m, promSink}, nil
}

// StartServer exposes the metrics
func (m *Sink) StartServer() error {
	http.Handle(m.path, promhttp.Handler())

	err := make(chan error)
	timeout := time.After(500 * time.Millisecond)

	// start the server in the background but wait to
	// check that it can bind correctly
	// if not return an error
	go func() {
		err <- http.ListenAndServe(fmt.Sprintf("%s:%d", m.addr, m.port), nil)
	}()

	select {
	case <-timeout:
		return nil
	case e := <-err:
		return e
	}

}

func (m *Sink) ServiceStarting() {
	m.m.IncrCounter([]string{"starting"}, 1)
}
