package httptest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setupPlugin(t *testing.T) *Plugin {
	p, err := New(nil, nil)
	require.NoError(t, err)

	return p
}

func TestValidatesOK(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configValid))

	require.NoError(t, err)
}

func TestValidatesPath(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configInvalidPath))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMissingPath(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configMissingPath))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestValidatesMethod(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configInvalidMethod))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesMissingMethod(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configMissingMethod))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidMethod.Error())
}

func TestValidatesInterval(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configInvalidInterval))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesMissingInterval(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configMissingInterval))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidInterval.Error())
}

func TestValidatesDuration(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configInvalidDuration))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidDuration.Error())
}

func TestValidatesMissingDuration(t *testing.T) {
	p := setupPlugin(t)

	err := p.Configure([]byte(configMissingDuration))

	require.Error(t, err)
	require.Contains(t, err.Error(), ErrInvalidDuration.Error())
}

var configValid = `
{
	"path": "/",
	"method": "GET",
	"payload": "{\"foo\": \"bar\"}",
	"interval": "10s",
	"duration": "10s"
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
