package clients

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sethvargo/go-retry"
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

	api := v1.NewAPI(c)

	var value model.Value
	var warn v1.Warnings
	var queryErr error

	// define a max retry duration
	ctxQuery, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// if there is an error when attempting the query, retry, the server might be temporarily unavailable
	queryErr = retry.Fibonacci(ctxQuery, 1*time.Second, func(ctx context.Context) error {
		// has the max duration elapsed
		if ctx.Err() != nil {
			if os.IsTimeout(ctx.Err()) {
				return fmt.Errorf("timeout while trying to query prometheus server: %s", address)
			}

			return ctx.Err()
		}

		value, warn, err = api.Query(ctx, query, ts)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("error querying prometheus server: %s, %s", address, err))
		}

		return nil
	})

	return value, warn, queryErr
}
