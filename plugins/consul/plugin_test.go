package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

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

	mc.On("DeleteServiceDefaults", mock.Anything).Return(nil)
	mc.On("DeleteServiceResolver", mock.Anything).Return(nil)
	mc.On("DeleteServiceRouter", mock.Anything).Return(nil)
	mc.On("DeleteServiceSplitter", mock.Anything).Return(nil)

	data := testutils.GetTestData(t, "valid_kubernetes_release.json")
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

func TestConfigureReturnsValidationErrors(t *testing.T) {
	p := &Plugin{}

	err := p.Configure([]byte{})
	require.Error(t, err)

	require.Contains(t, err.Error(), ErrConsulService.Error())
}

func TestSetupCreatesConsulServiceDefaults(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceDefaults", "api")
}

func TestSetupFailsOnCreateServiceDefaultsError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceDefaults")
	mc.On("CreateServiceDefaults", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background())

	mc.AssertCalled(t, "CreateServiceDefaults", "api")

	require.Error(t, err)
}

func TestSetupCreatesConsulServiceResolver(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceResolver", "api")
}

func TestSetupFailsOnCreateServiceResolverError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceResolver")
	mc.On("CreateServiceResolver", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background())
	require.Error(t, err)
}

func TestSetupCreatesConsulServiceRouter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceRouter", "api")
}

func TestSetupFailsOnCreateServiceRouterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceRouter")
	mc.On("CreateServiceRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background())
	require.Error(t, err)
}

func TestScaleUpdatesServiceSplitter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Scale(context.Background(), 80)
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceSplitter", "api", 20, 80)
}

func TestScaleReturnsErrorOnUpdateError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceSplitter")
	mc.On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Scale(context.Background(), 80)
	require.Error(t, err)
}

func TestDestroyDeletesServiceRouter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceRouter", "api")
}

func TestDestroyFailsOnDeleteServiceRouterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteServiceRouter")
	mc.On("DeleteServiceRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())
	require.Error(t, err)
}

func TestDestroyDeletesConsulServiceSplitter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceSplitter", "api")
}

func TestDestroyFailsOnDeletesConsulServiceSplitterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteServiceSplitter")
	mc.On("DeleteServiceSplitter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())
	require.Error(t, err)
}

func TestDestroyCreatesConsulServiceResolver(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceResolver", "api")
}

func TestDeleteFailsOnCreateServiceResolverError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteServiceResolver")
	mc.On("DeleteServiceResolver", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())
	require.Error(t, err)
}

func TestDestroyDeletesConsulServiceDefaults(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceDefaults", "api")
}

func TestDestroyFailsOnDeleteServiceDefaultsError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteServiceDefaults")
	mc.On("DeleteServiceDefaults", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())

	mc.AssertCalled(t, "DeleteServiceDefaults", "api")

	require.Error(t, err)
}
