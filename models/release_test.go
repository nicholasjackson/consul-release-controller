package models

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupDeployment(t *testing.T) (*Release, *mocks.Mocks) {
	d := &Release{}

	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_release.json"))
	d.FromJsonBody(ioutil.NopCloser(data))

	mp, pm := mocks.BuildMocks(t)

	// build the deployment
	err := d.Build(mp)
	require.NoError(t, err)

	mp.AssertCalled(t, "CreateReleaser", d.Releaser.Name)
	pm.ReleaserMock.AssertCalled(t, "Configure", d.Releaser.Config)

	mp.AssertCalled(t, "CreateRuntime", d.Runtime.Name)
	pm.RuntimeMock.AssertCalled(t, "Configure", d.Runtime.Config)

	mp.AssertCalled(t, "CreateMonitor", d.Monitor.Name)
	pm.MonitorMock.AssertCalled(t, "Configure", d.Monitor.Config)

	mp.AssertCalled(t, "CreateStrategy", d.Strategy.Name)
	pm.StrategyMock.AssertCalled(t, "Configure", d.Name, d.Namespace, d.Strategy.Config)

	return d, pm
}

func historyContains(r *Release, state string) bool {
	for _, s := range r.StateHistory {
		if s.State == state {
			return true
		}
	}

	return false
}

func TestBuildSetsUpPluginsAndState(t *testing.T) {
	mp, _ := mocks.BuildMocks(t)

	// test vanilla
	d := &Release{}
	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_release.json"))
	d.FromJsonBody(ioutil.NopCloser(data))
	d.Build(mp)

	require.Equal(t, StateStart, d.State())

	// test with existing state
	d = &Release{}
	data = bytes.NewBuffer(testutils.GetTestData(t, "idle_kubernetes_release.json"))
	d.FromJsonBody(ioutil.NopCloser(data))
	d.Build(mp)

	require.Equal(t, StateIdle, d.State())
}

func TestToJsonSerializesState(t *testing.T) {
	mp, _ := mocks.BuildMocks(t)
	d := &Release{}
	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_release.json"))
	d.FromJsonBody(ioutil.NopCloser(data))
	d.Build(mp)

	releaseJson := d.ToJson()
	require.Contains(t, string(releaseJson), `"current_state":"state_start"`)
}

func TestDeployWithNoErrorDeploysAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateStart)
	d.Deploy()

	require.Eventually(t, func() bool { return historyContains(d, StateConfigure) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "Deploy", mock.Anything, mock.Anything)
}

func TestDeployWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "Deploy")
	pm.RuntimeMock.On("Deploy", mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateStart)
	d.Deploy()

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventDeployedWithNoErrorSetsupReleaseAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateDeploy)
	d.state.Event(EventDeployed)

	require.Eventually(t, func() bool { return historyContains(d, StateConfigure) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything)
}

func TestEventDeployedWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Setup")
	pm.ReleaserMock.On("Setup", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateDeploy)
	d.state.Event(EventDeployed)

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventConfiguredWithNoErrorSetsInitialTrafficAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateConfigure)
	d.state.Event(EventConfigured, -1)

	require.Eventually(t, func() bool { return historyContains(d, StateScale) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, -1)
}

func TestEventConfiguredWithScaleErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateConfigure)
	d.state.Event(EventConfigured)

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventScaledWithNoErrorMonitorsAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateScale)
	d.state.Event(EventScaled)

	require.Eventually(t, func() bool { return historyContains(d, StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)

	// check that the traffic has been sent
	pm.StrategyMock.AssertCalled(t, "Execute", mock.Anything)
}

func TestEventScaledWithUnhealthyMontiorRollsback(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusFail, -1, nil)

	d.state.SetState(StateScale)
	d.state.Event(EventScaled)

	require.Eventually(t, func() bool { return historyContains(d, StateRollback) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateRollback }, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventScaledWithMontiorErrorRollsback(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusFail, -1, fmt.Errorf("boom"))

	d.state.SetState(StateScale)
	d.state.Event(EventScaled)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateFail }, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventScaledWithCompletePromotes(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.StrategyMock.Mock, "Execute")
	pm.StrategyMock.On("Execute", mock.Anything).Return(interfaces.StrategyStatusComplete, 100, nil)

	d.state.SetState(StateScale)
	d.state.Event(EventScaled)

	require.Eventually(t, func() bool { return historyContains(d, StatePromote) }, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventHealthyWithNoErrorScalesAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy, 10)

	require.Eventually(t, func() bool { return historyContains(d, StateScale) }, 100*time.Millisecond, 1*time.Millisecond)

	// check that the traffic has been sent
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 10)
}

func TestEventHealthyWithNoTrafficFails(t *testing.T) {
	d, _ := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateFail }, 100*time.Millisecond, 1*time.Millisecond)
}

func TestEventHealthyWithScaleErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventHealthy, 20)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateFail }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 20)
}

func TestEventCompleteWithNoErrorPromotesAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StatePromote) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "Promote", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventCompleteWithPromoteErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "Promote")
	pm.RuntimeMock.On("Promote", mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateFail }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "Promote", mock.Anything)
	pm.RuntimeMock.AssertNotCalled(t, "Scale", mock.Anything)
}

func TestEventCompleteWithScaleErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Scale")
	pm.ReleaserMock.On("Scale", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateMonitor)
	d.state.Event(EventComplete)

	require.Eventually(t, func() bool { return historyContains(d, StateFail) }, 100*time.Millisecond, 1*time.Millisecond)
	require.Eventually(t, func() bool { return d.CurrentState == StateFail }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "Promote", mock.Anything)
	pm.ReleaserMock.AssertCalled(t, "Scale", mock.Anything, 0)
}

func TestEventPromotedWithNoErrorCallsPluginAndMovesState(t *testing.T) {
	d, _ := setupDeployment(t)

	d.state.SetState(StatePromote)
	d.state.Event(EventPromoted)

	require.Eventually(t, func() bool { return historyContains(d, StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)

}
