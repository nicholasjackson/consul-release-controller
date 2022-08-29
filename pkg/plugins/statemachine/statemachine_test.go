package statemachine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
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
	pm.ReleaserMock.AssertCalled(t, "Configure", r.Releaser.Config, mock.Anything, mock.Anything)

	pp.AssertCalled(t, "CreateRuntime", r.Runtime.Name)
	pm.RuntimeMock.AssertCalled(t, "Configure", r.Runtime.Config, mock.Anything, mock.Anything)

	pp.AssertCalled(t, "CreateMonitor", r.Monitor.Name, r.Name, pm.RuntimeMock.BaseConfig().Namespace, r.Runtime.Name)
	pm.MonitorMock.AssertCalled(t, "Configure", r.Monitor.Config, mock.Anything, mock.Anything)

	pp.AssertCalled(t, "CreateStrategy", r.Strategy.Name)
	pm.StrategyMock.AssertCalled(t, "Configure", r.Strategy.Config, mock.Anything, mock.Anything)

	pp.AssertCalled(t, "CreateWebhook", r.Webhooks[0].Name)
	pm.WebhookMock.AssertCalled(t, "Configure", r.Webhooks[0].Config, mock.Anything, mock.Anything)

	pp.AssertCalled(t, "CreatePostDeploymentTest", r.PostDeploymentTest.Name, pm.ReleaserMock.BaseConfig().ConsulService, "", r.Runtime.Name, pm.MonitorMock)
	pm.PostDeploymentMock.AssertCalled(t, "Configure", r.PostDeploymentTest.Config, mock.Anything, mock.Anything)

	t.Cleanup(func() {
		if t.Failed() {
			fmt.Println(pm.LogBuffer.String())
		}
	})

	return r, sm, pm
}

func historyContains(r *models.Release, state string) bool {
	for _, s := range r.StateHistory() {
		if s.State == state {
			return true
		}
	}

	return false
}

func TestEventConfigureWithSetupErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Setup")
	pm.ReleaserMock.On("Setup", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventConfigureWithInitErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything, mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertNotCalled(t, "WaitUntilServiceHealthy", mock.Anything, pm.RuntimeMock.PrimarySubsetFilter())
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventConfigureWithHealthCheckErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "WaitUntilServiceHealthy")
	pm.ReleaserMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "WaitUntilServiceHealthy", mock.Anything, pm.RuntimeMock.PrimarySubsetFilter())
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventConfigureWithScaleErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventConfigureWithRemoveErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventConfigureWithNoErrorSetsStatusIdle(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateStart)
	sm.Event(interfaces.EventConfigure)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything, mock.Anything)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)

	// ensure webhook dispatched
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithInitErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything, mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithHealthCheckErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "WaitUntilServiceHealthy")
	pm.ReleaserMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "WaitUntilServiceHealthy", mock.Anything, pm.RuntimeMock.PrimarySubsetFilter())
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithScaleErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithRemoveErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithNoPrimarySetsStatusMonitor(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "InitPrimary")
	pm.RuntimeMock.On("InitPrimary", mock.Anything, mock.Anything).Return(interfaces.RuntimeDeploymentNoAction, nil)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertNotCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployWithNoErrorSetsStatusIdle(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDeploy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "InitPrimary", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployedWithPostDeploymentTestErrorSetsStatusRollback(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.PostDeploymentMock.Mock, "Execute")
	pm.PostDeploymentMock.On("Execute", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateRollback) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.PostDeploymentMock.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
	pm.StrategyMock.AssertNotCalled(t, "Execute", mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployedWithExecuteErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything, mock.Anything).Return(interfaces.StrategyStatusFailed, 0, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployedWithExecuteSuccessSetsStatusScale(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything, mock.Anything).Return(interfaces.StrategyStatusSuccess, 20, nil)

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateScale) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDeployedWithExecuteCompleteSetsStatusScale(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything, mock.Anything).Return(interfaces.StrategyStatusComplete, 100, nil)

	sm.SetState(interfaces.StateDeploy)
	sm.Event(interfaces.EventDeployed)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StatePromote) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
}

func TestEventHealthyWithNoTrafficSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, mock.Anything)
}

func TestEventHealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventHealthyWithNoScaleErrorSetsStatusMonitor(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithScaleCandidateErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithPromoteErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "PromoteCandidate")
	pm.RuntimeMock.On("PromoteCandidate", mock.Anything).Return(interfaces.RuntimeDeploymentInternalError, fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithHealthCheckErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "WaitUntilServiceHealthy")
	pm.ReleaserMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.ReleaserMock.AssertCalled(t, "WaitUntilServiceHealthy", mock.Anything, pm.RuntimeMock.PrimarySubsetFilter())
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithScalePrimaryErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(nil)
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventCompleteWithNoErrorSetsStatusIdle(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventComplete)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "PromoteCandidate", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventUnhealthyWithScaleErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 0).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventUnhealthyRemoveCandidateErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemoveCandidate")
	pm.RuntimeMock.On("RemoveCandidate", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventUnhealthyWithNoErrorSetsStatusIdle(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateMonitor)
	sm.Event(interfaces.EventUnhealthy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
	pm.RuntimeMock.AssertCalled(t, "RemoveCandidate", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithRestoreOriginalErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RestoreOriginal")
	pm.RuntimeMock.On("RestoreOriginal", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithHealthCheckErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "WaitUntilServiceHealthy")
	pm.ReleaserMock.On("WaitUntilServiceHealthy", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "WaitUntilServiceHealthy", mock.Anything, pm.RuntimeMock.CandidateSubsetFilter())
	pm.ReleaserMock.AssertNotCalled(t, "Scale", mock.Anything, 100)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithScaleErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, 100).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithRemovePrimaryErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "RemovePrimary")
	pm.RuntimeMock.On("RemovePrimary", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithDestroyErrorSetsStatusFail(t *testing.T) {
	r, sm, pm := setupTests(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Destroy")
	pm.ReleaserMock.On("Destroy", mock.Anything).Return(fmt.Errorf("boom"))

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}

func TestEventDestroyWithNoErrorSetsStatusIdle(t *testing.T) {
	r, sm, pm := setupTests(t)

	sm.SetState(interfaces.StateIdle)
	sm.Event(interfaces.EventDestroy)

	require.Eventually(t, func() bool { return historyContains(r, interfaces.StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "RestoreOriginal", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 100)
	pm.RuntimeMock.AssertCalled(t, "RemovePrimary", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Destroy", mock.Anything)
	pm.WebhookMock.AssertCalled(t, "Send", mock.Anything)
}
