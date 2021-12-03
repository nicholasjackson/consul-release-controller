package models

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupDeployment(t *testing.T) (*Deployment, *plugins.Mocks) {
	d := &Deployment{}

	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_deployment.json"))
	d.FromJsonBody(ioutil.NopCloser(data))

	mp, pm := plugins.BuildMocks(t)

	// build the deployment
	err := d.Build(mp)
	require.NoError(t, err)

	pm.ReleaserMock.AssertCalled(t, "Configure", d.Releaser.Config)
	pm.RuntimeMock.AssertCalled(t, "Configure", d.Runtime.Config)

	return d, pm
}

func TestInitializeWithNoErrorCallsPluginAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)
	d.Configure()

	require.Eventually(t, func() bool { return d.StateIs(StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.ReleaserMock.AssertCalled(t, "Setup", mock.Anything, mock.Anything)
}

func TestInitializeWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.ReleaserMock.Mock, "Setup")
	pm.ReleaserMock.On("Setup", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.Configure()

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestDeployWithNoErrorCallsPluginAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateIdle)
	d.Deploy()

	require.Eventually(t, func() bool { return d.StateIs(StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.RuntimeMock.AssertCalled(t, "Deploy", mock.Anything, mock.Anything)
}

func TestDeployWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.RuntimeMock.Mock, "Deploy")
	pm.RuntimeMock.On("Deploy", mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateIdle)
	d.Deploy()

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}
