package canary

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T, config string) (*Plugin, *mocks.MonitorMock) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	p, _ := New(m.MonitorMock)

	err := p.Configure([]byte(config), log, m.StoreMock)
	require.NoError(t, err)

	return p, m.MonitorMock
}

func TestConfigureSetsState(t *testing.T) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetState")
	m.StoreMock.On("GetState").Return([]byte(`{"candidate_traffic":10}`), nil)

	p, _ := New(m.MonitorMock)
	err := p.Configure([]byte(canaryStrategy), log, m.StoreMock)

	require.NoError(t, err)
	require.Equal(t, 10, p.state.CandidateTraffic)
}

func TestConfigureSetsDefaultStateOnError(t *testing.T) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	testutils.ClearMockCall(&m.StoreMock.Mock, "GetState")
	m.StoreMock.On("GetState").Return(nil, fmt.Errorf("boom"))

	p, _ := New(m.MonitorMock)
	err := p.Configure([]byte(canaryStrategy), log, m.StoreMock)

	require.NoError(t, err)
	require.Equal(t, -1, p.state.CandidateTraffic)
}

func TestSetsIntitialDelayToIntervalWhenNotSet(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategyWithoutInitialDelay)
	require.Equal(t, p.config.InitialDelay, p.config.Interval)
}

func TestValidatesConfig(t *testing.T) {
	log := hclog.NewNullLogger()
	_, m := mocks.BuildMocks(t)

	p, _ := New(m.MonitorMock)

	err := p.Configure([]byte(canaryStrategyWithValidationErrors), log, m.StoreMock)
	require.Error(t, err)

	require.Contains(t, err.Error(), ErrInvalidInitialDelay.Error())
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
	require.Contains(t, err.Error(), ErrTrafficStep.Error())
	require.Contains(t, err.Error(), ErrMaxTraffic.Error())
	require.Contains(t, err.Error(), ErrThreshold.Error())
}

func TestSetsInitialTrafficAndReturnsFirstRun(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategy)

	status, traffic, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(status))
	require.Equal(t, 10, traffic)

	mm.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
}

func TestSetsInitialTrafficToTrafficStepWhenNotSetAndReturnsFirstRun(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategyWithoutInitialTraffic)

	status, traffic, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(status))
	require.Equal(t, 20, traffic)

	mm.AssertNotCalled(t, "Check", mock.Anything)
}

func TestCallsMonitorCheckAndReturnsWhenNoError(t *testing.T) {
	st := time.Now()
	p, mm := setupPlugin(t, canaryStrategy)

	_, _, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	_, _, err = p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything, mock.Anything, 30*time.Millisecond)
	mm.AssertNumberOfCalls(t, "Check", 1)

	et := time.Since(st)
	require.Greater(t, et, 30*time.Millisecond, "Execute should sleep for interval before check")
}

func TestExecuteIncrementsTrafficSubsequentRuns(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)

	_, _, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	status, traffic, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusSuccess, string(status))
	require.Equal(t, 30, traffic)
}

func TestExecuteReturnsCompleteWhenAllChecksComplete(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.state.CandidateTraffic = 70

	status, traffic, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusComplete, string(status))
	require.Equal(t, 100, traffic)
}

func TestReturnsErrorWhenChecksFail(t *testing.T) {
	p, mm := setupPlugin(t, canaryStrategy)
	testutils.ClearMockCall(&mm.Mock, "Check")
	p.state.CandidateTraffic = 10

	mm.On("Check", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(interfaces.CheckFailed, fmt.Errorf("boom"))

	status, traffic, err := p.Execute(context.Background(), "test-deployment")
	require.NoError(t, err)

	require.Equal(t, interfaces.StrategyStatusFailed, string(status))
	require.Equal(t, traffic, 0)

	// should call check 5 times due to error threshold
	mm.AssertNumberOfCalls(t, "Check", 5)
}

func TestGetPrimaryTrafficReturns100WhenMinusOne(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)

	traf := p.GetPrimaryTraffic()

	require.Equal(t, 100, traf)
}

func TestGetPrimaryTrafficReturns0WhenGreater100(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.state.CandidateTraffic = 101

	traf := p.GetPrimaryTraffic()

	require.Equal(t, 0, traf)
}

func TestGetPrimaryTrafficReturnsCorrectValue(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.state.CandidateTraffic = 20

	traf := p.GetPrimaryTraffic()

	require.Equal(t, 80, traf)
}

func TestGetCandidateTrafficReturnsCorrectValue(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.state.CandidateTraffic = 20

	traf := p.GetCandidateTraffic()

	require.Equal(t, 20, traf)
}

func TestGetCandidateTrafficReturns0WhenMinusOne(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)

	traf := p.GetCandidateTraffic()

	require.Equal(t, 0, traf)
}

func TestGetCandidateTrafficReturns100WhenGreater100(t *testing.T) {
	p, _ := setupPlugin(t, canaryStrategy)
	p.state.CandidateTraffic = 101

	traf := p.GetCandidateTraffic()

	require.Equal(t, 100, traf)
}

const canaryStrategyWithoutInitialDelay = `
{
  "interval": "30ms",
  "initial_traffic": 10,
  "traffic_step": 20,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategy = `
{
  "interval": "30ms",
  "initial_traffic": 10,
  "initial_delay": "30ms",
  "traffic_step": 20,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategyWithoutInitialTraffic = `
{
  "interval": "30ms",
  "traffic_step": 20,
  "max_traffic": 90,
  "error_threshold": 5
}
`

const canaryStrategyWithValidationErrors = `
{
  "initial_delay": "acs",
  "interval": "30",
  "initial_traffic": 101,
  "traffic_step": 1100,
  "max_traffic": -3,
  "error_threshold": -1
}
`
