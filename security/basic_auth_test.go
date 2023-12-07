// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package security

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type secTestKey uint8

const (
	original secTestKey = iota
	extra
	reason
)

const (
	wisdom            = "The man who is swimming against the stream knows the strength of it."
	extraWisdom       = "Our greatest glory is not in never falling, but in rising every time we fall."
	expReason         = "I like the dreams of the future better than the history of the past."
	authenticatedPath = "/blah"
	testPassword      = "123456"
	basicPrincipal    = "admin"
)

var basicAuthHandler = UserPassAuthentication(func(user, pass string) (interface{}, error) {
	if user == basicPrincipal && pass == testPassword {
		return basicPrincipal, nil
	}
	return "", errors.Unauthenticated("basic")
})

func TestValidBasicAuth(t *testing.T) {
	ba := BasicAuth(basicAuthHandler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)

	req.SetBasicAuth(basicPrincipal, testPassword)
	ok, usr, err := ba.Authenticate(req)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, basicPrincipal, usr)
}

func TestInvalidBasicAuth(t *testing.T) {
	ba := BasicAuth(basicAuthHandler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)
	req.SetBasicAuth(basicPrincipal, basicPrincipal)

	ok, usr, err := ba.Authenticate(req)
	require.Error(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", usr)

	assert.NotEmpty(t, FailedBasicAuth(req))
	assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
}

func TestMissingbasicAuth(t *testing.T) {
	ba := BasicAuth(basicAuthHandler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)

	ok, usr, err := ba.Authenticate(req)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, nil, usr)

	assert.NotEmpty(t, FailedBasicAuth(req))
	assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
}

func TestNoRequestBasicAuth(t *testing.T) {
	ba := BasicAuth(basicAuthHandler)

	ok, usr, err := ba.Authenticate("token")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, usr)
}

var basicAuthHandlerCtx = UserPassAuthenticationCtx(func(ctx context.Context, user, pass string) (context.Context, interface{}, error) {
	if user == basicPrincipal && pass == testPassword {
		return context.WithValue(ctx, extra, extraWisdom), basicPrincipal, nil
	}
	return context.WithValue(ctx, reason, expReason), "", errors.Unauthenticated("basic")
})

func TestValidBasicAuthCtx(t *testing.T) {
	ba := BasicAuthCtx(basicAuthHandlerCtx)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), original, wisdom))

	req.SetBasicAuth(basicPrincipal, testPassword)
	ok, usr, err := ba.Authenticate(req)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, basicPrincipal, usr)
	assert.Equal(t, wisdom, req.Context().Value(original))
	assert.Equal(t, extraWisdom, req.Context().Value(extra))
	assert.Nil(t, req.Context().Value(reason))
}

func TestInvalidBasicAuthCtx(t *testing.T) {
	ba := BasicAuthCtx(basicAuthHandlerCtx)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), original, wisdom))
	req.SetBasicAuth(basicPrincipal, basicPrincipal)

	ok, usr, err := ba.Authenticate(req)
	require.Error(t, err)
	assert.True(t, ok)
	assert.Equal(t, "", usr)
	assert.Equal(t, wisdom, req.Context().Value(original))
	assert.Nil(t, req.Context().Value(extra))
	assert.Equal(t, expReason, req.Context().Value(reason))
}

func TestMissingbasicAuthCtx(t *testing.T) {
	ba := BasicAuthCtx(basicAuthHandlerCtx)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authenticatedPath, nil)
	require.NoError(t, err)
	req = req.WithContext(context.WithValue(req.Context(), original, wisdom))

	ok, usr, err := ba.Authenticate(req)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, nil, usr)

	assert.Equal(t, wisdom, req.Context().Value(original))
	assert.Nil(t, req.Context().Value(extra))
	assert.Nil(t, req.Context().Value(reason))
}

func TestNoRequestBasicAuthCtx(t *testing.T) {
	ba := BasicAuthCtx(basicAuthHandlerCtx)

	ok, usr, err := ba.Authenticate("token")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, usr)
}
