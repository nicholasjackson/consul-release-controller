package models

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/testutils"
	"github.com/stretchr/testify/require"
)

func setupDeployment(t *testing.T) (*Release, *mocks.Mocks) {
	StepDelay = 1 * time.Millisecond

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
	pm.MonitorMock.AssertCalled(t, "Configure", "api-deployment", "default", d.Runtime.Name, d.Monitor.Config)

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

	require.Equal(t, StateStart, d.CurrentState)

	// test with existing state
	d = &Release{}
	data = bytes.NewBuffer(testutils.GetTestData(t, "idle_kubernetes_release.json"))
	d.FromJsonBody(ioutil.NopCloser(data))
	d.Build(mp)

	require.Equal(t, StateIdle, d.CurrentState)
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
