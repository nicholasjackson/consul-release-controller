package mocks

import (
	"encoding/json"

	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
	"github.com/stretchr/testify/mock"
)

type WebhookMock struct {
	mock.Mock
}

func (m *WebhookMock) Configure(c json.RawMessage) error {
	args := m.Called(c)

	return args.Error(0)
}

func (m *WebhookMock) Send(msg interfaces.WebhookMessage) error {
	args := m.Called(msg)

	return args.Error(0)
}
