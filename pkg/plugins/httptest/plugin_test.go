package httptest

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/mocks"
	"github.com/nicholasjackson/consul-release-controller/pkg/testutils"
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
	mm.On("Check", mock.Anything, mock.Anything, 30*time.Second).Return(interfaces.CheckSuccess, nil)

	p, err := New("test", "testnamespace", "kubernetes", mm)
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

	err := p.Configure([]byte(configValid), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.NoError(t, err)
}

func TestValidatesPath(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidPath), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMissingPath(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingPath), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMethod(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidMethod), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesMissingMethod(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingMethod), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesInterval(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidInterval), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesMissingInterval(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingInterval), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesDuration(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidTimeout), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidTimeout.Error())
}

func TestValidatesMissingDuration(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingTimeout), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidTimeout.Error())
}

func TestValidatesRequiredTestPasses(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configInvalidTestPasses), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidTestPasses.Error())
}

func TestValidatesMissingTestPasses(t *testing.T) {
	p, _, _ := setupPlugin(t)

	err := p.Configure([]byte(configMissingTestPasses), hclog.NewNullLogger(), &mocks.StoreMock{})

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidTestPasses.Error())
}

func TestExecuteCallsExternalServer(t *testing.T) {
	p, _, hr := setupPlugin(t)

	err := p.Configure([]byte(configValid), hclog.NewNullLogger(), &mocks.StoreMock{})
	require.NoError(t, err)

	err = p.Execute(context.TODO(), "test-deployment")
	require.NoError(t, err)

	require.Len(t, *hr, 5)
	require.Equal(t, "/", (*hr)[0].Path)
	require.Equal(t, "GET", (*hr)[0].Method)
	require.Equal(t, "test.testnamespace", (*hr)[0].Host)
	require.Equal(t, "{\"foo\": \"bar\"}", string((*hr)[0].Body))
}

func TestExecuteCallsCheck5Times(t *testing.T) {
	p, mm, _ := setupPlugin(t)

	err := p.Configure([]byte(configValid), hclog.NewNullLogger(), &mocks.StoreMock{})
	require.NoError(t, err)

	err = p.Execute(context.TODO(), "test-deployment")
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything, mock.Anything, 30*time.Second)
	mm.AssertNumberOfCalls(t, "Check", 5)
}

func TestExecuteCallsCheck8TimesWhenOneTestFails(t *testing.T) {
	p, mm, _ := setupPlugin(t)
	testutils.ClearMockCall(&mm.Mock, "Check")

	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckFailed, fmt.Errorf("oops"))
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Once().Return(interfaces.CheckSuccess, nil)

	err := p.Configure([]byte(configValid), hclog.NewNullLogger(), &mocks.StoreMock{})
	require.NoError(t, err)

	err = p.Execute(context.TODO(), "test-deployment")
	require.NoError(t, err)

	mm.AssertCalled(t, "Check", mock.Anything, mock.Anything, 30*time.Second)
	mm.AssertNumberOfCalls(t, "Check", 8)
}

func TestExecuteTimesoutIfNoSuccess(t *testing.T) {
	p, mm, _ := setupPlugin(t)
	testutils.ClearMockCall(&mm.Mock, "Check")
	mm.On("Check", mock.Anything, mock.Anything, mock.Anything).Return(interfaces.CheckFailed, fmt.Errorf("oops"))

	err := p.Configure([]byte(configValid), hclog.NewNullLogger(), &mocks.StoreMock{})
	require.NoError(t, err)

	err = p.Execute(context.TODO(), "test-deployment")
	require.Error(t, err)
	require.Contains(t, err.Error(), "timeout")
}

var configValid = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10ns",
	"required_test_passes": 5,
	"timeout": "10ms"
}
`

var configInvalidPath = `
{
	"path": "'",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 5,
	"timeout": "10s"
}
`

var configMissingPath = `
{
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 5,
	"timeout": "10s"
}
`

var configInvalidMethod = `
{
	"path": "/",
	"method": "GIT",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 5,
	"timeout": "10s"
}
`

var configMissingMethod = `
{
	"path": "/",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 5,
	"timeout": "10s"
}
`

var configInvalidInterval = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10",
	"required_test_passes": 5,
	"timeout": "10s"
}
`
var configMissingInterval = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"required_test_passes": 5,
	"timeout": "10s"
}
`

var configInvalidTimeout = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"duration": "10",
	"required_test_passes": 5,
	"timeout": "10"
}
`

var configMissingTimeout = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 5
}
`

var configInvalidTestPasses = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"required_test_passes": 0,
	"timeout": "10s"
}
`

var configMissingTestPasses = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"timeout": "10s"
}
`
