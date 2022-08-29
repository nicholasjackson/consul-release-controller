package slack

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockClient struct {
	mock.Mock
}

func (s *mockClient) Send(url, content string) error {
	args := s.Called(url, content)

	return args.Error(0)
}

func setupTests(t *testing.T, config string) (*Plugin, *mockClient) {
	p, _ := New()

	err := p.Configure([]byte(config), hclog.NewNullLogger(), &mocks.StoreMock{})
	assert.NoError(t, err)

	mc := &mockClient{}
	mc.On("Send", mock.Anything, mock.Anything).Return(nil)
	p.client = mc

	return p, mc
}

func TestValidatesURL(t *testing.T) {
	p, _ := New()
	err := p.Configure([]byte(configWithMissingURL), hclog.NewNullLogger(), &mocks.StoreMock{})

	assert.Error(t, err)
}

func TestConfiguresWithoutError(t *testing.T) {
	p, _ := New()
	err := p.Configure([]byte(validConfig), hclog.NewNullLogger(), &mocks.StoreMock{})

	assert.NoError(t, err)
}

func TestSendsMessageWithDefaultContentNoError(t *testing.T) {
	p, mc := setupTests(t, validConfig)

	p.Send(interfaces.WebhookMessage{
		Name:      "testname",
		Namespace: "testnamespace",
		Title:     "testtitle",
		Outcome:   "testoutcome",
		State:     "teststate",
		Error:     "",
	})

	mc.AssertCalled(t, "Send", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	content := args.Get(1).(string)

	assert.Contains(t, content, `has changed to "teststate"`)
	assert.Contains(t, content, `The outcome is "testoutcome"`)
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

	mc.AssertCalled(t, "Send", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	content := args.Get(1).(string)

	assert.Contains(t, content, `has changed to "teststate"`)
	assert.Contains(t, content, `An error occurred when processing: It went boom`)
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

	mc.AssertCalled(t, "Send", mock.Anything, mock.Anything)

	args := mc.Calls[0].Arguments
	content := args.Get(1).(string)

	assert.Contains(t, content, `my template teststate`)
}

func TestDoesNotSendWhenStatusNotMatching(t *testing.T) {
	p, mc := setupTests(t, validConfigWithStatus)

	p.Send(interfaces.WebhookMessage{
		Name:      "testname",
		Namespace: "testnamespace",
		Title:     "testtitle",
		Outcome:   "testoutcome",
		State:     "teststate",
		Error:     "",
	})

	mc.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

var configWithMissingURL = `
{
	"template": "abcdef"
}
`

var validConfig = `
{
	"url": "abcdef"
}
`

var validConfigWithTemplate = `
{
	"url": "abcdef",
	"template": "my template {{ .State }}"
}
`

var validConfigWithStatus = `
{
	"url": "abcdef",
	"status": ["state_destroy"]
}
`
