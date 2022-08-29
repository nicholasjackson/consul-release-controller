package consul

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/clients"
	"github.com/nicholasjackson/consul-release-controller/pkg/models"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testSetupStorage(t *testing.T) (*Storage, *models.Release, *clients.ConsulMock) {
	l := hclog.NewNullLogger()
	s, _ := NewStorage(l)

	mc := &clients.ConsulMock{}

	s.consulClient = mc

	td := testutils.GetTestData(t, "valid_kubernetes_release.json")
	r := &models.Release{}
	json.Unmarshal(td, r)

	return s, r, mc
}

func TestUpsertReleaseSavesToConsul(t *testing.T) {
	// check serializes release to JSON and stores in KV
	s, r, mc := testSetupStorage(t)

	mc.On("SetKV", mock.Anything, mock.Anything).Return(nil)

	err := s.UpsertRelease(r)
	require.NoError(t, err)

	mc.AssertCalled(t, "SetKV", "consul-release-controller/releases/api/config", mock.Anything)
}

func TestUpsertReleaseReturnsErrorOnConsulError(t *testing.T) {
	// check serializes release to JSON and stores in KV
	s, r, mc := testSetupStorage(t)

	mc.On("SetKV", mock.Anything, mock.Anything).Return(fmt.Errorf("boom"))

	err := s.UpsertRelease(r)
	require.Error(t, err)
}

func TestListReleasesReturnsAllReleases(t *testing.T) {
	s, _, mc := testSetupStorage(t)

	mc.On("ListKV", mock.Anything).Return([]string{
		"consul-release-controller/releases/api/config",
		"consul-release-controller/releases/payments/plugin-state/runtime",
		"consul-release-controller/releases/payments/plugin-state/monitor",
		"consul-release-controller/releases/payments/config",
		"consul-release-controller/releases/currency/config",
	}, nil)
	mc.On("GetKV", mock.Anything, mock.Anything).Return([]byte(`{"name": "api"}`), nil)

	rels, err := s.ListReleases(nil)
	assert.NoError(t, err)

	mc.AssertCalled(t, "ListKV", "consul-release-controller/releases")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/config")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/payments/config")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/currency/config")

	require.Len(t, rels, 3)
}

func TestListReleasesReturnsFilteredReleases(t *testing.T) {
	s, _, mc := testSetupStorage(t)
	mc.On("ListKV", mock.Anything).Return([]string{
		"consul-release-controller/releases/api/config",
		"consul-release-controller/releases/payments/plugin-state/runtime",
		"consul-release-controller/releases/payments/plugin-state/monitor",
		"consul-release-controller/releases/payments/config",
		"consul-release-controller/releases/currency/config",
	}, nil)

	mc.On("GetKV", mock.Anything, mock.Anything).Once().Return([]byte(`{"name": "api", "runtime": {"plugin_name": "nomad"}}`), nil)
	mc.On("GetKV", mock.Anything, mock.Anything).Once().Return([]byte(`{"name": "api", "runtime": {"plugin_name": "kubernetes"}}`), nil)
	mc.On("GetKV", mock.Anything, mock.Anything).Once().Return([]byte(`{"name": "api", "runtime": {"plugin_name": "ecs"}}`), nil)

	rels, err := s.ListReleases(&interfaces.ListOptions{Runtime: "kubernetes"})
	assert.NoError(t, err)

	mc.AssertCalled(t, "ListKV", "consul-release-controller/releases")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/config")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/payments/config")
	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/currency/config")

	require.Len(t, rels, 1)
}

func TestGetReleaseReturnsRelease(t *testing.T) {
	s, _, mc := testSetupStorage(t)
	mc.On("GetKV", mock.Anything, mock.Anything).Return([]byte(`{"name": "api"}`), nil)

	rel, err := s.GetRelease("api")
	assert.NoError(t, err)
	require.NotNil(t, rel)

	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/config")
}

func TestGetReleaseReturnsNotFound(t *testing.T) {
	s, _, mc := testSetupStorage(t)
	mc.On("GetKV", mock.Anything, mock.Anything).Return(nil, nil)

	rel, err := s.GetRelease("api")
	assert.Error(t, err)
	assert.Nil(t, rel)

	require.Equal(t, interfaces.ReleaseNotFound, err)

	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/config")
}

func TestDeleteReleaseDeletesRelease(t *testing.T) {
	s, _, mc := testSetupStorage(t)
	mc.On("DeleteKV", mock.Anything, mock.Anything).Return(nil)

	err := s.DeleteRelease("api")
	assert.NoError(t, err)

	mc.AssertCalled(t, "DeleteKV", "consul-release-controller/releases/api")
}

func TestUpsertStateSetsState(t *testing.T) {
	s, r, mc := testSetupStorage(t)
	ps := s.CreatePluginStateStore(r, "test-plugin")

	mc.On("SetKV", mock.Anything, mock.Anything).Return(nil)

	err := ps.UpsertState([]byte("testing"))
	require.NoError(t, err)

	mc.AssertCalled(t, "SetKV", "consul-release-controller/releases/api/plugin-state/test-plugin", mock.Anything)
}

func TestGetStateGetsState(t *testing.T) {
	s, r, mc := testSetupStorage(t)
	ps := s.CreatePluginStateStore(r, "test-plugin")

	mc.On("GetKV", mock.Anything, mock.Anything).Return([]byte("data"), nil)

	d, err := ps.GetState()
	require.NoError(t, err)
	require.Equal(t, []byte("data"), d)

	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/plugin-state/test-plugin", mock.Anything)
}

func TestGetStateRetrunsErrorWhenNoState(t *testing.T) {
	s, r, mc := testSetupStorage(t)
	ps := s.CreatePluginStateStore(r, "test-plugin")

	mc.On("GetKV", mock.Anything, mock.Anything).Return(nil, nil)

	d, err := ps.GetState()
	require.Error(t, err)
	require.Equal(t, interfaces.PluginStateNotFound, err)
	require.Nil(t, d)

	mc.AssertCalled(t, "GetKV", "consul-release-controller/releases/api/plugin-state/test-plugin", mock.Anything)
}
