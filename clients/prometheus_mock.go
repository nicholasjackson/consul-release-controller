package clients

import (
	"context"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/mock"
)

// PrometheusMock is a mock implementation of the Queryable interface for testing
type PrometheusMock struct {
	mock.Mock
}

func (mq *PrometheusMock) Query(ctx context.Context, address, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	args := mq.Called(ctx, query, ts)

	if mv, ok := args.Get(0).(model.Value); ok {
		return mv, args.Get(1).(v1.Warnings), args.Error(2)
	}

	return nil, args.Get(1).(v1.Warnings), args.Error(2)
}
