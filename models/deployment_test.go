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

type pluginMocks struct {
	sp *plugins.SetupMock
	rm *plugins.RuntimeMock
}

func setupDeployment(t *testing.T) (*Deployment, *pluginMocks) {
	d := &Deployment{}

	data := bytes.NewBuffer(testutils.GetTestData(t, "valid_kubernetes_deployment.json"))
	d.FromJsonBody(ioutil.NopCloser(data))

	// create the mock plugins
	sm := &plugins.SetupMock{}
	sm.On("Configure", mock.Anything).Return(nil)
	sm.On("Setup", mock.Anything, mock.Anything).Return(nil)

	rm := &plugins.RuntimeMock{}
	rm.On("Configure", mock.Anything).Return(nil)
	rm.On("Deploy", mock.Anything, mock.Anything).Return(nil)

	mp := &plugins.ProviderMock{}
	mp.On("CreateReleaser", mock.Anything).Return(sm, nil)
	mp.On("CreateRuntime", mock.Anything).Return(rm, nil)

	// build the deployment
	err := d.Build(mp)
	require.NoError(t, err)

	sm.AssertCalled(t, "Configure", d.Setup.Config)
	rm.AssertCalled(t, "Configure", d.Deployment.Config)

	return d, &pluginMocks{sm, rm}
}

func TestInitializeWithNoErrorCallsPluginAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)
	d.Initialize()

	require.Eventually(t, func() bool { return d.StateIs(StateIdle) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.sp.AssertCalled(t, "Setup", mock.Anything, mock.Anything)
}

func TestInitializeWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.sp.Mock, "Setup")
	pm.sp.On("Setup", mock.Anything, mock.Anything).Return(fmt.Errorf("Boom"))

	d.Initialize()

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}

func TestDeployWithNoErrorCallsPluginAndMovesState(t *testing.T) {
	d, pm := setupDeployment(t)

	d.state.SetState(StateIdle)
	d.Deploy()

	require.Eventually(t, func() bool { return d.StateIs(StateMonitor) }, 100*time.Millisecond, 1*time.Millisecond)
	pm.rm.AssertCalled(t, "Deploy", mock.Anything, mock.Anything)
}

func TestDeployWithErrorDoesNotMoveState(t *testing.T) {
	d, pm := setupDeployment(t)

	testutils.ClearMockCall(&pm.rm.Mock, "Deploy")
	pm.rm.On("Deploy", mock.Anything).Return(fmt.Errorf("Boom"))

	d.state.SetState(StateIdle)
	d.Deploy()

	require.Eventually(t, func() bool {
		return d.StateIs(StateFail)
	}, 100*time.Millisecond, 1*time.Millisecond)
}
