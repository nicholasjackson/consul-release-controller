package discord

import (
	"testing"

	"github.com/DisgoOrg/disgo/discord"
	"github.com/DisgoOrg/disgo/rest"
	"github.com/DisgoOrg/disgo/webhook"
	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/interfaces"
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

func TestSendsMessageWithDefaultContent(t *testing.T) {
	p, mc := setupTests(t, validConfig)

	p.Send(interfaces.WebhookMessage{
		Name:      "testname",
		Namespace: "testnamespace",
		Title:     "testtitle",
		Outcome:   "testoutcome",
		State:     "teststate",
		Error:     "",
	})

	mc.AssertCalled(t, "CreateEmbeds", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	embed := args.Get(0).([]discord.Embed)

	assert.Equal(t, "testtitle", embed[0].Title)
	assert.Contains(t, embed[0].Description, `has changed to "teststate"`)
}

func TestSendsMessageWithCustomContent(t *testing.T) {
	p, mc := setupTests(t, validConfigWithTemplate)

	p.Send(interfaces.WebhookMessage{
		Name:      "testname",
		Namespace: "testnamespace",
		Title:     "testtitle",
		Outcome:   "testoutcome",
		State:     "teststate",
		Error:     "",
	})

	mc.AssertCalled(t, "CreateEmbeds", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	embed := args.Get(0).([]discord.Embed)

	assert.Equal(t, "testtitle", embed[0].Title)
	assert.Contains(t, embed[0].Description, `my template teststate`)
}

func TestSendsMessageWithDefaultContentError(t *testing.T) {
	p, mc := setupTests(t, validConfig)

	p.Send(interfaces.WebhookMessage{
		Name:      "testname",
		Namespace: "testnamespace",
		Title:     "testtitle",
		Outcome:   "testoutcome",
		State:     "teststate",
		Error:     "It went boom",
	})

	mc.AssertCalled(t, "CreateEmbeds", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	embed := args.Get(0).([]discord.Embed)

	assert.Equal(t, "testtitle", embed[0].Title)
	assert.Contains(t, embed[0].Description, `An error occurred when processing: It went boom`)
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

var validConfigWithTemplate = `
{
	"token": "abcdef",
	"id": "abcdef",
	"template": "my template {{ .State }}"
}
`
