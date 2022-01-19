package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestEventConfigureWithSetupErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Setup")
	pm.ReleaserMock.On("Setup", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateStart)
	d.state.Event(EventConfigure)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
}

func TestEventConfigureWithInitErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	d.state.SetState(StateStart)
	d.state.Event(EventConfigure)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
}

func TestEventConfigureWithScaleErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	d.state.SetState(StateStart)
	d.state.Event(EventConfigure)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventConfigureWithRemoveErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateStart)
	d.state.Event(EventConfigure)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventConfigureWithNoErrorSetsStatusIdle(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateStart)
	d.state.Event(EventConfigure)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithInitErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDeploy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
}

func TestEventDeployWithScaleErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDeploy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventDeployWithRemoveErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDeploy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithNoPrimarySetsStatusMonitor(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentNoAction, nil)

	d.state.SetState(StateIdle)
	d.state.Event(EventDeploy)

	require.Eventually(t, func() bool { return historyContains(d, StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertNotCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithNoErrorSetsStatusIdle(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateIdle)
	d.state.Event(EventDeploy)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployedWithExecuteErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusFail, 0, fmt.Errorf("boom"))

	d.state.SetState(StateDeploy)
	d.state.Event(EventDeployed)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
}

func TestEventDeployedWithExecuteSuccessSetsStatusScale(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusSuccess, 20, nil)

	d.state.SetState(StateDeploy)
	d.state.Event(EventDeployed)

	require.Eventually(t, func() bool { return historyContains(d, StateScale) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventDeployedWithExecuteCompleteSetsStatusScale(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusComplete, 100, nil)

	d.state.SetState(StateDeploy)
	d.state.Event(EventDeployed)

	require.Eventually(t, func() bool { return historyContains(d, StatePromote) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
}

func TestEventHealthyWithNoTrafficSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, mock.Anything)
}

func TestEventHealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventHealthyWithNoScaleErrorSetsStatusMonitor(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(d, StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventCompleteWithScaleCandidateErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
}

func TestEventCompleteWithPromoteErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "PromoteCandidate")
	pm.RuntimeMock.On("PromoteCandidate", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
}

func TestEventCompleteWithScalePrimaryErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(nil)
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventCompleteWithRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventCompleteWithNoErrorSetsStatusIdle(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventUnhealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventUnhealthyRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventUnhealthyWithNoErrorSetsStatusIdle(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDestroyWithRestoreOriginalErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RestoreOriginal")
	pm.RuntimeMock.On("RestoreOriginal", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDestroy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
}

func TestEventDestroyWithScaleErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDestroy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
}

func TestEventDestroyWithRemovePrimaryErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemovePrimary")
	pm.RuntimeMock.On("RemovePrimary", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDestroy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
}

func TestEventDestroyWithDestroyErrorSetsStatusFail(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Destroy")
	pm.ReleaserMock.On("Destroy", mock.Anything).Return(fmt.Errorf("boom"))

	d.state.SetState(StateIdle)
	d.state.Event(EventDestroy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
}

func TestEventDestroyWithNoErrorSetsStatusIdle(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateIdle)
	d.state.Event(EventDestroy)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
}
