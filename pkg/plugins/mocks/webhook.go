package mocks

import (
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type WebhookMock struct {
	mock.Mock
}

func (m *WebhookMock) Configure(data json.RawMessage, log hclog.Logger, store interfaces.PluginStateStore) error {
	args := m.Called(data, log, store)

	return args.Error(0)
}

func (m *WebhookMock) Send(msg interfaces.WebhookMessage) error {
	args := m.Called(msg)

	return args.Error(0)
}
