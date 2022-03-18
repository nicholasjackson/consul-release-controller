package httptest

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/plugins/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type httpRequest struct {
	Headers http.Header
	Host    string
	Path    string
	Body    []byte
	Method  string
}

func setupPlugin(t *testing.T) (*Plugin, *mocks.MonitorMock, *[]*httpRequest) {
	mm := &mocks.MonitorMock{}

	p, err := New("test", "testnamespace", "kubernetes", hclog.NewNullLogger(), mm)
	require.NoError(t, err)

	reqs := []*httpRequest{}

	// start the test server
	s := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			hr := &httpRequest{}
			hr.Path = r.URL.Path
			hr.Body, _ = ioutil.ReadAll(r.Body)
			hr.Headers = r.Header
			hr.Method = r.Method
			hr.Host = r.Host

			reqs = append(reqs, hr)
		}))

	t.Setenv("UPSTREAMS", s.URL)

	t.Cleanup(func() {
		s.Close()
	})

	return p, mm, &reqs
}

func TestValidatesOK(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configValid))

	require.NoError(t, err)
}

func TestValidatesPath(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidPath))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMissingPath(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingPath))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMethod(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidMethod))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesMissingMethod(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingMethod))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesInterval(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidInterval))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesMissingInterval(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingInterval))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesDuration(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidDuration))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidDuration.Error())
}

func TestValidatesMissingDuration(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingDuration))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidDuration.Error())
}

func TestExecuteCallsExternalServer(t *testing.T) {
	p, _, hr := setupPlugin(t)

	err := p.Configure([]byte(configValid))
	require.NoError(t, err)

	err = p.Execute(context.TODO())
	require.NoError(t, err)

	require.Len(t, *hr, 1)
	require.Equal(t, "/", (*hr)[0].Path)
	require.Equal(t, "GET", (*hr)[0].Method)
	require.Equal(t, "test.testnamespace", (*hr)[0].Host)
	require.Equal(t, "{\"foo\": \"bar\"}", string((*hr)[0].Body))
}

func TestExecuteCallsCheck(t *testing.T) {
	p, mm, _ := setupPlugin(t)

	err := p.Configure([]byte(configValid))
	require.NoError(t, err)

	err = p.Execute(context.TODO())
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything, 10*time.Nanosecond)
}

var configValid = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10ns",
	"duration": "10ns"
}
`

var configInvalidPath = `
{
	"path": "'",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10s"
}
`

var configMissingPath = `
{
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10s"
}
`

var configInvalidMethod = `
{
	"path": "/",
	"method": "GIT",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10s"
}
`

var configMissingMethod = `
{
	"path": "/",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10s"
}
`

var configInvalidInterval = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10",
	"duration": "10s"
}
`
var configMissingInterval = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"duration": "10s"
}
`

var configInvalidDuration = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10"
}
`

var configMissingDuration = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s"
}
`
