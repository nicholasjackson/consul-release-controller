package mocks

import (
	"encoding/json"

	"github.com/stretchr/testify/mock"
)

type WebhookMock struct {
	mock.Mock
}

func (m *WebhookMock) Configure(c json.RawMessage) error {
	args := m.Called(c)

	return args.Error(0)
}

func (m *WebhookMock) Send(title, content string) error {
	args := m.Called(title, content)

	return args.Error(0)
}
