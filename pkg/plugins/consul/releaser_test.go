package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T) (*Plugin, *clients.ConsulMock) {
	// ensure the sync delay is set to a value for testing
	syncDelay = 1 * time.Millisecond

	log := hclog.NewNullLogger()
	mc := &clients.ConsulMock{}

	mc.On("CreateServiceDefaults", mock.Anything).Return(nil)
	mc.On("CreateServiceResolver", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mc.On("CreateUpstreamRouter", mock.Anything, mock.Anything).Return(nil)
	mc.On("CreateServiceSplitter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mc.On("CreateServiceIntention", mock.Anything, mock.Anything).Return(nil)

	mc.On("DeleteServiceSplitter", mock.Anything).Return(nil)
	mc.On("DeleteServiceDefaults", mock.Anything).Return(nil)
	mc.On("DeleteServiceResolver", mock.Anything).Return(nil)
	mc.On("DeleteUpstreamRouter", mock.Anything).Return(nil)
	mc.On("DeleteServiceIntention", mock.Anything).Return(nil)

	data := testutils.GetTestData(t, "valid_kubernetes_release.json")
	dep := map[string]interface{}{}
	json.Unmarshal(data, &dep)

	conf := dep["releaser"].(map[string]interface{})["config"]
	jsn, err := json.Marshal(conf)
	assert.NoError(t, err)

	p, _ := New()
	err = p.Configure(jsn, log, &mocks.StoreMock{})
	assert.NoError(t, err)

	p.consulClient = mc

	return p, mc
}

func TestSerializesConfig(t *testing.T) {
	p, _ := setupPlugin(t)

	assert.Equal(t, "api", p.config.ConsulService)
}

func TestConfigureReturnsValidationErrors(t *testing.T) {
	p := &Plugin{}

	err := p.Configure([]byte{}, hclog.NewNullLogger(), &mocks.StoreMock{})
	require.Error(t, err)

	require.Contains(t, err.Error(), ErrConsulService.Error())
}

func TestSetupCreatesConsulServiceDefaults(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), "primary", "candidate")
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceDefaults", "api")
	mc.AssertCalled(t, "CreateServiceDefaults", clients.ControllerServiceName)
	mc.AssertCalled(t, "CreateServiceDefaults", clients.UpstreamRouterName)
}

func TestSetupFailsOnCreateServiceDefaultsForServiceError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceDefaults")
	mc.On("CreateServiceDefaults", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")

	mc.AssertCalled(t, "CreateServiceDefaults", "api")
	mc.AssertNotCalled(t, "CreateServiceDefaults", clients.ControllerServiceName)
	mc.AssertNotCalled(t, "CreateServiceDefaults", clients.UpstreamRouterName)

	require.Error(t, err)
}

func TestSetupFailsOnCreateServiceDefaultsForControllerError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceDefaults")
	mc.On("CreateServiceDefaults", mock.Anything).Once().Return(nil)
	mc.On("CreateServiceDefaults", mock.Anything).Once().Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")

	mc.AssertCalled(t, "CreateServiceDefaults", "api")
	mc.AssertCalled(t, "CreateServiceDefaults", clients.ControllerServiceName)
	mc.AssertNotCalled(t, "CreateServiceDefaults", clients.UpstreamRouterName)

	require.Error(t, err)
}

func TestSetupFailsOnCreateServiceDefaultsForUpstreamError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceDefaults")
	mc.On("CreateServiceDefaults", mock.Anything).Once().Return(nil)
	mc.On("CreateServiceDefaults", mock.Anything).Once().Return(nil)
	mc.On("CreateServiceDefaults", mock.Anything).Once().Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")

	mc.AssertCalled(t, "CreateServiceDefaults", "api")
	mc.AssertCalled(t, "CreateServiceDefaults", clients.ControllerServiceName)
	mc.AssertCalled(t, "CreateServiceDefaults", clients.UpstreamRouterName)

	require.Error(t, err)
}

func TestSetupCreatesConsulServiceResolver(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), "primary", "candidate")
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceResolver", "api", "primary", "candidate")
}

func TestSetupFailsOnCreateServiceResolverError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceResolver")
	mc.On("CreateServiceResolver", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")
	require.Error(t, err)
}

func TestSetupCreatesUpstreamServiceRouter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), "primary", "candidate")
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateUpstreamRouter", "api")
}

func TestSetupFailsOnCreateUpstreamServiceRouterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateUpstreamRouter")
	mc.On("CreateUpstreamRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")
	require.Error(t, err)
}

func TestSetupCreatesServiceIntention(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Setup(context.Background(), "primary", "candidate")
	require.NoError(t, err)

	mc.AssertCalled(t, "CreateServiceIntention", "api")
}

func TestSetupFailsOnCreateServiceIntentionError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "CreateServiceIntention")
	mc.On("CreateServiceIntention", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Setup(context.Background(), "primary", "candidate")
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

func TestDestroyDeletesServiceSplitter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceSplitter", "api")
}

func TestDestroyFailsOnDeletesServiceSplitterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteServiceSplitter")
	mc.On("DeleteServiceSplitter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())
	require.Error(t, err)
}

func TestDestroyDeletesUpstreamServiceRouter(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteUpstreamRouter", "api")
}

func TestDestroyFailsOnDeleteUpstreamServiceRouterError(t *testing.T) {
	p, mc := setupPlugin(t)

	testutils.ClearMockCall(&mc.Mock, "DeleteUpstreamRouter")
	mc.On("DeleteUpstreamRouter", mock.Anything).Return(fmt.Errorf("boom"))

	err := p.Destroy(context.Background())
	require.Error(t, err)
}

func TestDestroyDeletesConsulServiceResolver(t *testing.T) {
	p, mc := setupPlugin(t)

	err := p.Destroy(context.Background())
	require.NoError(t, err)

	mc.AssertCalled(t, "DeleteServiceResolver", "api")
}

func TestDeleteFailsOnDeleteServiceResolverError(t *testing.T) {
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
