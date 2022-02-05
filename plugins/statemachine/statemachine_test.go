package statemachine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-release-controller/models"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTests(t *testing.T) (*models.Release, *StateMachine, *mocks.Mocks) {
	stepDelay = 1 * time.Millisecond

	pp, pm := mocks.BuildMocks(t)
	r := &models.Release{}
	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_release.json"))
	r.FromJsonBody(ioutil.NopCloser(data))

	sm, err := New(r, pp)
	require.NoError(t, err)

	pp.AssertCalled(t, "CreateReleaser", r.Releaser.Name)
	pm.ReleaserMock.AssertCalled(t, "Configure", r.Releaser.Config)

	pp.AssertCalled(t, "CreateRuntime", r.Runtime.Name)
	pm.RuntimeMock.AssertCalled(t, "Configure", r.Runtime.Config)

	pp.AssertCalled(t, "CreateMonitor", r.Monitor.Name)
	pm.MonitorMock.AssertCalled(t, "Configure", "api-deployment", "default", r.Runtime.Name, r.Monitor.Config)

	pp.AssertCalled(t, "CreateStrategy", r.Strategy.Name)
	pm.StrategyMock.AssertCalled(t, "Configure", r.Name, r.Namespace, r.Strategy.Config)

	return r, sm, pm
}

func historyContains(sm *StateMachine, state string) bool {
	for _, s := range sm.StateHistory() {
		if s.State == state {
			return true
		}
	}

	return false
}

func TestEventConfigureWithSetupErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Setup")
	pm.ReleaserMock.On("Setup", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
}

func TestEventConfigureWithInitErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
}

func TestEventConfigureWithScaleErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventConfigureWithRemoveErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventConfigureWithNoErrorSetsStatusIdle(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithInitErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
}

func TestEventDeployWithScaleErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventDeployWithRemoveErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithNoPrimarySetsStatusMonitor(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything).Return(interfaces.RuntimeDeploymentNoAction, nil)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertNotCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployWithNoErrorSetsStatusIdle(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDeployedWithExecuteErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusFail, 0, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
}

func TestEventDeployedWithExecuteSuccessSetsStatusScale(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusSuccess, 20, nil)

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateScale) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventDeployedWithExecuteCompleteSetsStatusScale(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusComplete, 100, nil)

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StatePromote) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
}

func TestEventHealthyWithNoTrafficSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, mock.Anything)
}

func TestEventHealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventHealthyWithNoScaleErrorSetsStatusMonitor(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventCompleteWithScaleCandidateErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
}

func TestEventCompleteWithPromoteErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "PromoteCandidate")
	pm.RuntimeMock.On("PromoteCandidate", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
}

func TestEventCompleteWithScalePrimaryErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(nil)
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventCompleteWithRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventCompleteWithNoErrorSetsStatusIdle(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventUnhealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventUnhealthyRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventUnhealthyWithNoErrorSetsStatusIdle(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
}

func TestEventDestroyWithRestoreOriginalErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RestoreOriginal")
	pm.RuntimeMock.On("RestoreOriginal", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
}

func TestEventDestroyWithScaleErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
}

func TestEventDestroyWithRemovePrimaryErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemovePrimary")
	pm.RuntimeMock.On("RemovePrimary", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
}

func TestEventDestroyWithDestroyErrorSetsStatusFail(t *testing.T) {
	_, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Destroy")
	pm.ReleaserMock.On("Destroy", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
}

func TestEventDestroyWithNoErrorSetsStatusIdle(t *testing.T) {
	_, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(sm, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
}
