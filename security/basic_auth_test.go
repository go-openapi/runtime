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
	wisdom       = "The man who is swimming against the stream knows the strength of it."
	extraWisdom  = "Our greatest glory is not in never falling, but in rising every time we fall."
	expReason    = "I like the dreams of the future better than the history of the past."
	testPassword = "123456"
)

func TestBasicAuth(t *testing.T) {
	basicAuthHandler := UserPassAuthentication(func(user, pass string) (interface{}, error) {
		if user == principal && pass == testPassword {
			return principal, nil
		}
		return "", errors.Unauthenticated("basic")
	})
	ba := BasicAuth(basicAuthHandler)

	t.Run("with valid basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, testPassword)

		ok, usr, err := ba.Authenticate(req)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, principal, usr)
	})

	t.Run("with invalid basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := ba.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)

		assert.NotEmpty(t, FailedBasicAuth(req))
		assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
	})

	t.Run("with missing basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)

		ok, usr, err := ba.Authenticate(req)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, usr)

		assert.NotEmpty(t, FailedBasicAuth(req))
		assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
	})

	t.Run("basic auth without request", func(*testing.T) {
		ok, usr, err := ba.Authenticate("token")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, usr)
	})

	t.Run("with realm, invalid basic auth", func(t *testing.T) {
		br := BasicAuthRealm("realm", basicAuthHandler)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := br.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)
		assert.Equal(t, "realm", FailedBasicAuth(req))
	})

	t.Run("with empty realm, invalid basic auth", func(t *testing.T) {
		br := BasicAuthRealm("", basicAuthHandler)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := br.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)
		assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
	})
}

func TestBasicAuthCtx(t *testing.T) {
	basicAuthHandlerCtx := UserPassAuthenticationCtx(func(ctx context.Context, user, pass string) (context.Context, interface{}, error) {
		if user == principal && pass == testPassword {
			return context.WithValue(ctx, extra, extraWisdom), principal, nil
		}
		return context.WithValue(ctx, reason, expReason), "", errors.Unauthenticated("basic")
	})
	ba := BasicAuthCtx(basicAuthHandlerCtx)
	ctx := context.WithValue(context.Background(), original, wisdom)

	t.Run("with valid basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
		require.NoError(t, err)

		req.SetBasicAuth(principal, testPassword)
		ok, usr, err := ba.Authenticate(req)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, principal, usr)

		assert.Equal(t, wisdom, req.Context().Value(original))
		assert.Equal(t, extraWisdom, req.Context().Value(extra))
		assert.Nil(t, req.Context().Value(reason))
	})

	t.Run("with invalid basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := ba.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)

		assert.Equal(t, wisdom, req.Context().Value(original))
		assert.Nil(t, req.Context().Value(extra))
		assert.Equal(t, expReason, req.Context().Value(reason))
	})

	t.Run("with missing basic auth", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
		require.NoError(t, err)

		ok, usr, err := ba.Authenticate(req)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, usr)

		assert.Equal(t, wisdom, req.Context().Value(original))
		assert.Nil(t, req.Context().Value(extra))
		assert.Nil(t, req.Context().Value(reason))
	})

	t.Run("basic auth without request", func(*testing.T) {
		ok, usr, err := ba.Authenticate("token")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, usr)
	})

	t.Run("with realm, invalid basic auth", func(t *testing.T) {
		br := BasicAuthRealmCtx("realm", basicAuthHandlerCtx)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := br.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)
		assert.Equal(t, "realm", FailedBasicAuth(req))
	})

	t.Run("with empty realm, invalid basic auth", func(t *testing.T) {
		br := BasicAuthRealmCtx("", basicAuthHandlerCtx)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
		require.NoError(t, err)
		req.SetBasicAuth(principal, principal)

		ok, usr, err := br.Authenticate(req)
		require.Error(t, err)
		assert.True(t, ok)
		assert.Equal(t, "", usr)
		assert.Equal(t, DefaultRealmName, FailedBasicAuth(req))
	})
}
