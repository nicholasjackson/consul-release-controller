package canary

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T, config string) (*Plugin, *mocks.MonitorMock) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	p, _ := New(log, m.MonitorMock)

	err := p.Configure([]byte(config))
	require.NoError(t, err)

	return p, m.MonitorMock
}

func TestCallsMonitorCheckAndReturnsWhenNoError(t *testing.T) {
	st := time.Now()
	p, mm := setupPlugin(t, canaryStrategy)

	_, _, err := p.Execute(context.Background())
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything)

	et := time.Since(st)
	require.Greater(t, et, 30*time.Millisecond, "Execute should sleep for interval before check")
}

func TestExecuteReturnsInitialTrafficFirstRun(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(state))
	require.Equal(t, 10, traffic)
}

func TestExecuteReturnsIncrementsTrafficSubsequentRuns(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 10

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(state))
	require.Equal(t, 20, traffic)
}

func TestExecuteReturnsCompleteWhenAllChecksComplete(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.currentTraffic = 80

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusComplete, string(state))
	require.Equal(t, 100, traffic)
}

func TestReturnsErrorWhenChecksFail(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategy)
	testutils.ClearMockCall(&mm.Mock, "Check")

	mm.On("Check", mock.Anything).Return(fmt.Errorf("boom"))

	state, traffic, err := p.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusFail, string(state))
	require.Equal(t, traffic, 0)

	// should call check 5 times due to error threshold
	mm.AssertNumberOfCalls(t, "Check", 5)
}

const canaryStrategy = `
{
  "interval": "30ms",
  "initial_traffic": 10,
  "traffic_step": 10,
  "max_traffic": 90,
  "error_threshold": 5
}
`
