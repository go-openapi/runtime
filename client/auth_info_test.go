// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net/http"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/runtime"
)

func TestBasicAuth(t *testing.T) {
	r := newRequest(http.MethodGet, "/", nil)

	writer := BasicAuth("someone", "with a password")
	err := writer.AuthenticateRequest(r, nil)
	require.NoError(t, err)

	req := new(http.Request)
	req.Header = make(http.Header)
	req.Header.Set(runtime.HeaderAuthorization, r.header.Get(runtime.HeaderAuthorization))
	usr, pw, ok := req.BasicAuth()
	require.True(t, ok)
	assert.Equal(t, "someone", usr)
	assert.Equal(t, "with a password", pw)
}

func TestAPIKeyAuth_Query(t *testing.T) {
	r := newRequest(http.MethodGet, "/", nil)

	writer := APIKeyAuth("api_key", "query", "the-shared-key")
	err := writer.AuthenticateRequest(r, nil)
	require.NoError(t, err)

	assert.Equal(t, "the-shared-key", r.query.Get("api_key"))
}

func TestAPIKeyAuth_Header(t *testing.T) {
	r := newRequest(http.MethodGet, "/", nil)

	writer := APIKeyAuth("X-Api-Token", "header", "the-shared-key")
	err := writer.AuthenticateRequest(r, nil)
	require.NoError(t, err)

	assert.Equal(t, "the-shared-key", r.header.Get("X-Api-Token"))
}

func TestBearerTokenAuth(t *testing.T) {
	r := newRequest(http.MethodGet, "/", nil)

	writer := BearerToken("the-shared-token")
	err := writer.AuthenticateRequest(r, nil)
	require.NoError(t, err)

	assert.Equal(t, "Bearer the-shared-token", r.header.Get(runtime.HeaderAuthorization))
}

func TestCompose(t *testing.T) {
	r := newRequest(http.MethodGet, "/", nil)

	writer := Compose(APIKeyAuth("X-Api-Key", "header", "the-api-key"), APIKeyAuth("X-Secret-Key", "header", "the-secret-key"))
	err := writer.AuthenticateRequest(r, nil)
	require.NoError(t, err)

	assert.Equal(t, "the-api-key", r.header.Get("X-Api-Key"))
	assert.Equal(t, "the-secret-key", r.header.Get("X-Secret-Key"))
}
