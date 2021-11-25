package models

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupDeployment() *Deployment {
	log := hclog.NewNullLogger()
	nm := &metrics.Null{}
	mc := &clients.MockConsul{}

	cl := &Clients{
		Consul: mc,
	}

	mc.On("CreateServiceDefaults", mock.Anything).Return(nil)
	mc.On("CreateServiceResolver", mock.Anything).Return(nil)
	mc.On("CreateServiceRouter", mock.Anything).Return(nil)
	mc.On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	d := NewDeployment(log, nm, cl)
	d.ConsulService = "testing"

	return d
}

func clearMockCall(mc *clients.MockConsul, method string) {
	calls := mc.ExpectedCalls
	new := []*mock.Call{}

	for _, c := range calls {
		if c.Method != method {
			new = append(new, c)
		}
	}

	mc.ExpectedCalls = new
}

func TestInitializeCreatesConsulServiceDefaults(t *testing.T) {
	d := setupDeployment()

	err := d.Initialize()
	require.NoError(t, err)

	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceDefaults", "cc-testing-primary")
	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceDefaults", "cc-testing-canary")

	require.True(t, d.StateIs(EventInitialized))
}

func TestInitializeFailsOnCreateServiceDefaultsError(t *testing.T) {
	d := setupDeployment()

	clearMockCall(d.clients.Consul.(*clients.MockConsul), "CreateServiceDefaults")
	d.clients.Consul.(*clients.MockConsul).On("CreateServiceDefaults", mock.Anything).Return(fmt.Errorf("boom"))

	err := d.Initialize()

	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceDefaults", "cc-testing-primary")
	d.clients.Consul.(*clients.MockConsul).AssertNotCalled(t, "CreateServiceDefaults", "cc-testing-canary")

	require.Error(t, err)

	require.False(t, d.StateIs(EventInitialized))
}

func TestInitializeCreatesConsulServiceResolver(t *testing.T) {
	d := setupDeployment()

	err := d.Initialize()
	require.NoError(t, err)

	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceResolver", "cc-testing")

	require.True(t, d.StateIs(EventInitialized))
}

func TestInitializeFailsOnCreateServiceResolverError(t *testing.T) {
	d := setupDeployment()

	clearMockCall(d.clients.Consul.(*clients.MockConsul), "CreateServiceResolver")
	d.clients.Consul.(*clients.MockConsul).On("CreateServiceResolver", mock.Anything).Return(fmt.Errorf("boom"))

	err := d.Initialize()
	require.Error(t, err)

	require.False(t, d.StateIs(EventInitialized))
}

func TestInitializeCreatesConsulServiceRouter(t *testing.T) {
	d := setupDeployment()

	err := d.Initialize()
	require.NoError(t, err)

	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceRouter", "cc-testing")

	require.True(t, d.StateIs(EventInitialized))
}

func TestInitializeFailsOnCreateServiceRouterError(t *testing.T) {
	d := setupDeployment()

	clearMockCall(d.clients.Consul.(*clients.MockConsul), "CreateServiceRouter")
	d.clients.Consul.(*clients.MockConsul).On("CreateServiceRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := d.Initialize()
	require.Error(t, err)

	require.False(t, d.StateIs(EventInitialized))
}

func TestInitializeCreatesConsulServiceSplitter(t *testing.T) {
	d := setupDeployment()

	err := d.Initialize()
	require.NoError(t, err)

	d.clients.Consul.(*clients.MockConsul).AssertCalled(t, "CreateServiceSplitter", "cc-testing", 100, 0)

	require.True(t, d.StateIs(EventInitialized))
}

func TestInitializeFailsOnCreateServiceSplitter(t *testing.T) {
	d := setupDeployment()

	clearMockCall(d.clients.Consul.(*clients.MockConsul), "CreateServiceSplitter")
	d.clients.Consul.(*clients.MockConsul).On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := d.Initialize()
	require.Error(t, err)

	require.False(t, d.StateIs(EventInitialized))
}
