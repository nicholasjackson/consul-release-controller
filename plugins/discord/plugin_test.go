package discord

import (
	"testing"

	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgo/rest"
	"github.com/DisgoOrg/disgo/webhook"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock
}

func (h *mockClient) CreateEmbeds(embeds []discord.Embed, opts ...rest.RequestOpt) (*webhook.Message, error) {
	args := h.Called(embeds, opts)

	return nil, args.Error(0)
}

func setupTests(t *testing.T, config string) (*Plugin, *mockClient) {
	p, _ := New(hclog.NewNullLogger())

	err := p.Configure([]byte(config))
	assert.NoError(t, err)

	mc := &mockClient{}
	mc.On("CreateEmbeds", mock.Anything, mock.Anything).Return(nil)
	p.client = mc

	return p, mc
}

func TestValidatesID(t *testing.T) {
	p, _ := New(hclog.NewNullLogger())
	err := p.Configure([]byte(configWithMissingID))

	assert.Error(t, err)
}

func TestValidatesToken(t *testing.T) {
	p, _ := New(hclog.NewNullLogger())
	err := p.Configure([]byte(configWithMissingToken))

	assert.Error(t, err)
}

func TestConfiguresWithoutError(t *testing.T) {
	p, _ := New(hclog.NewNullLogger())
	err := p.Configure([]byte(validConfig))

	assert.NoError(t, err)
}

func TestSendsMessage(t *testing.T) {
	p, mc := setupTests(t, validConfig)

	p.Send("my title", "my content string")

	mc.AssertCalled(t, "CreateEmbeds", mock.Anything, mock.Anything)
}

var configWithMissingID = `
{
	"token": "abcdef"
}
`

var configWithMissingToken = `
{
	"id": "abcdef"
}
`

var validConfig = `
{
	"token": "abcdef",
	"id": "abcdef"
}
`
