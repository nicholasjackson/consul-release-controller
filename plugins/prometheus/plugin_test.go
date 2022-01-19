package prometheus

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/clients"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T, config string) (*Plugin, *clients.PrometheusMock) {
	l := hclog.NewNullLogger()
	p, _ := New(l)

	pm := &clients.PrometheusMock{}
	pm.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		model.Vector{
			&model.Sample{Value: 100, Timestamp: model.Time(time.Now().Unix())},
		},
		v1.Warnings{},
		nil,
	)

	err := p.Configure("api-deployment", "default", "kubernetes", []byte(config))
	require.NoError(t, err)

	p.client = pm

	return p, pm
}

func TestPluginReturnsErrorWhenPresetNotFound(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueriesInvalidPreset)

	err := p.Check(context.Background(), 30*time.Second)
	require.Error(t, err)

	pm.AssertNotCalled(t, "Query")
}

func TestPluginReturnsErrorWhenNilValue(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueriesInvalidPreset)

	testutils.ClearMockCall(&pm.Mock, "Query")
	pm.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		v1.Warnings{},
		nil,
	)

	err := p.Check(context.Background(), 30*time.Second)
	require.Error(t, err)

	pm.AssertNotCalled(t, "Query")
}

func TestPluginReturnsErrorWhenEmptyVector(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueriesInvalidPreset)

	testutils.ClearMockCall(&pm.Mock, "Query")
	pm.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		model.Vector{},
		v1.Warnings{},
		nil,
	)

	err := p.Check(context.Background(), 30*time.Second)
	require.Error(t, err)

	pm.AssertNotCalled(t, "Query")
}

func TestPluginReturnsErrorWhenQueryValueLessThanMin(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueries)

	testutils.ClearMockCall(&pm.Mock, "Query")
	pm.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		model.Vector{
			&model.Sample{Value: model.SampleValue(1), Timestamp: model.Time(time.Now().Unix())},
		},
		v1.Warnings{},
		nil,
	)

	err := p.Check(context.Background(), 30*time.Second)
	require.Error(t, err)

	pm.AssertNumberOfCalls(t, "Query", 1)
}

func TestPluginReturnsErrorWhenQueryValueGreaterThanMax(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueries)

	testutils.ClearMockCall(&pm.Mock, "Query")
	pm.On("Query", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		model.Vector{
			&model.Sample{Value: model.SampleValue(201), Timestamp: model.Time(time.Now().Unix())},
		},
		v1.Warnings{},
		nil,
	)

	err := p.Check(context.Background(), 30*time.Second)
	require.Error(t, err)

	pm.AssertNumberOfCalls(t, "Query", 2)
}

func TestPluginExecutesQueriesAndChecksValue(t *testing.T) {
	p, pm := setupPlugin(t, twoDefaultQueries)

	err := p.Check(context.Background(), 30*time.Second)
	require.NoError(t, err)

	pm.AssertNumberOfCalls(t, "Query", 2)

	// check that the query interpolation was added correctly
	call1Args := pm.Calls[0].Arguments[1]
	call2Args := pm.Calls[1].Arguments[1]

	require.Contains(t, call1Args, `namespace="default"`)
	require.Contains(t, call1Args, `pod=~"api-deployment-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"`)
	require.Contains(t, call1Args, `[30s]`)

	require.Contains(t, call2Args, `namespace="default"`)
	require.Contains(t, call2Args, `pod=~"api-deployment-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"`)
	require.Contains(t, call2Args, `[30s]`)
}

const twoDefaultQueries = `
{
	"address": "http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090",
	"queries": [
	  {
	    "name": "request-success",
	    "preset": "envoy-request-success",
	    "min":99
	  },
	  {
	    "name": "request-duration",
	    "preset": "envoy-request-duration",
	    "min":20,
	    "max": 200
	  }
	]
}
`

const twoDefaultQueriesInvalidPreset = `
{
	"address": "http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090",
	"queries": [
	  {
	    "name": "request-success",
	    "preset": "envoy-request-success",
	    "min":99
	  },
	  {
	    "name": "request-duration",
	    "preset": "not-found",
	    "min":20,
	    "max": 200
	  }
	]
}
`
