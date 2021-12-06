package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T) (*Plugin, *clients.ConsulMock) {
	log := hclog.NewNullLogger()
	mc := &clients.ConsulMock{}

	mc.On("CreateServiceDefaults", mock.Anything).Return(nil)
	mc.On("CreateServiceResolver", mock.Anything).Return(nil)
	mc.On("CreateServiceRouter", mock.Anything).Return(nil)
	mc.On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	data := testutils.GetTestData(t, "valid_kubernetes_deployment.json")
	dep := map[string]interface{}{}
	json.Unmarshal(data, &dep)

	conf := dep["releaser"].(map[string]interface{})["config"]
	jsn, err := json.Marshal(conf)
	assert.NoError(t, err)

	p := &Plugin{log: log, consulClient: mc}
	err = p.Configure(jsn)
	assert.NoError(t, err)

	return p, mc
}

func TestSerializesConfig(t *testing.T) {
	p, _ := setupPlugin(t)

	assert.Equal(t, "api", p.config.ConsulService)
}

func TestInitializeCreatesConsulServiceDefaults(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), func() {})
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceDefaults", "cc-api-primary")
	mc.AssertCalled(t, "CreateServiceDefaults", "cc-api-canary")
}

func TestInitializeFailsOnCreateServiceDefaultsError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceDefaults")
	mc.On("CreateServiceDefaults", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), func() {})

	mc.AssertCalled(t, "CreateServiceDefaults", "cc-api-primary")
	mc.AssertNotCalled(t, "CreateServiceDefaults", "cc-api-canary")

	require.Error(t, err)
}

func TestInitializeCreatesConsulServiceResolver(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), func() {})
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceResolver", "cc-api")
}

func TestInitializeFailsOnCreateServiceResolverError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceResolver")
	mc.On("CreateServiceResolver", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), func() {})
	require.Error(t, err)
}

func TestInitializeCreatesConsulServiceRouter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), func() {})
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceRouter", "cc-api")
}

func TestInitializeFailsOnCreateServiceRouterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceRouter")
	mc.On("CreateServiceRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), func() {})
	require.Error(t, err)
}

func TestInitializeCreatesConsulServiceSplitter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), func() {})
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceSplitter", "cc-api", 100, 0)
}

func TestInitializeFailsOnCreateServiceSplitter(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceSplitter")
	mc.On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), func() {})
	require.Error(t, err)
}

func TestInitializeSuccessCallsCallback(t *testing.T) {
	callbackCalled := true
	callback := func() {
		callbackCalled = true
	}

	p, _ := setupPlugin(t)

	err := p.Setup(context.Background(), callback)
	require.NoError(t, err)

	assert.Eventually(t, func() bool { return callbackCalled }, 100*time.Millisecond, 1*time.Millisecond)
}
