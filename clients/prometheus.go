package clients

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Prometheus defines an interface that can query a prometheus API
// this is cut down from the official v1.API to only include required fields
type Prometheus interface {
	// Query performs a query for the given time.
	Query(ctx context.Context, address, query string, ts time.Time) (model.Value, v1.Warnings, error)
}

type PrometheusImpl struct {
}

func NewPrometheus() (Prometheus, error) {
	return &PrometheusImpl{}, nil
}

func (p *PrometheusImpl) Query(ctx context.Context, address, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	// create the promethus client
	c, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		return nil, v1.Warnings{}, fmt.Errorf("unable to create new Prometheus client: %s", err)
	}

	v1.NewAPI(c)

	return nil, v1.Warnings{}, nil
}
