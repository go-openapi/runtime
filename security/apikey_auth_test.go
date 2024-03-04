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
	"fmt"
	"net/http"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	apiKeyParam  = "api_key"
	apiKeyHeader = "X-API-KEY"
)

func TestApiKeyAuth(t *testing.T) {
	tokenAuth := TokenAuthentication(func(token string) (interface{}, error) {
		if token == validToken {
			return principal, nil
		}
		return nil, errors.Unauthenticated("token")
	})

	t.Run("with invalid initialization", func(t *testing.T) {
		assert.Panics(t, func() { APIKeyAuth(apiKeyParam, "qery", tokenAuth) })
	})

	t.Run("with token in query param", func(t *testing.T) {
		ta := APIKeyAuth(apiKeyParam, query, tokenAuth)

		t.Run("with valid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, validToken), nil)
			require.NoError(t, err)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Equal(t, principal, usr)
			require.NoError(t, err)
		})

		t.Run("with invalid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, invalidToken), nil)
			require.NoError(t, err)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Nil(t, usr)
			require.Error(t, err)
		})

		t.Run("with missing token", func(t *testing.T) {
			// put the token in the header, but query param is expected
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, validToken)

			ok, usr, err := ta.Authenticate(req)
			assert.False(t, ok)
			assert.Nil(t, usr)
			require.NoError(t, err)
		})
	})

	t.Run("with token in header", func(t *testing.T) {
		ta := APIKeyAuth(apiKeyHeader, header, tokenAuth)

		t.Run("with valid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, validToken)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Equal(t, principal, usr)
			require.NoError(t, err)
		})

		t.Run("with invalid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, invalidToken)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Nil(t, usr)
			require.Error(t, err)
		})

		t.Run("with missing token", func(t *testing.T) {
			// put the token in the query param, but header is expected
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, validToken), nil)
			require.NoError(t, err)

			ok, usr, err := ta.Authenticate(req)
			assert.False(t, ok)
			assert.Nil(t, usr)
			require.NoError(t, err)
		})
	})
}

func TestApiKeyAuthCtx(t *testing.T) {
	tokenAuthCtx := TokenAuthenticationCtx(func(ctx context.Context, token string) (context.Context, interface{}, error) {
		if token == validToken {
			return context.WithValue(ctx, extra, extraWisdom), principal, nil
		}
		return context.WithValue(ctx, reason, expReason), nil, errors.Unauthenticated("token")
	})
	ctx := context.WithValue(context.Background(), original, wisdom)

	t.Run("with invalid initialization", func(t *testing.T) {
		assert.Panics(t, func() { APIKeyAuthCtx(apiKeyParam, "qery", tokenAuthCtx) })
	})

	t.Run("with token in query param", func(t *testing.T) {
		ta := APIKeyAuthCtx(apiKeyParam, query, tokenAuthCtx)

		t.Run("with valid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, validToken), nil)
			require.NoError(t, err)
			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Equal(t, principal, usr)
			require.NoError(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Equal(t, extraWisdom, req.Context().Value(extra))
			assert.Nil(t, req.Context().Value(reason))
		})

		t.Run("with invalid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, invalidToken), nil)
			require.NoError(t, err)
			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Nil(t, usr)
			require.Error(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Equal(t, expReason, req.Context().Value(reason))
			assert.Nil(t, req.Context().Value(extra))
		})

		t.Run("with missing token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, validToken)

			ok, usr, err := ta.Authenticate(req)
			assert.False(t, ok)
			assert.Nil(t, usr)
			require.NoError(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Nil(t, req.Context().Value(reason))
			assert.Nil(t, req.Context().Value(extra))
		})
	})

	t.Run("with token in header", func(t *testing.T) {
		ta := APIKeyAuthCtx(apiKeyHeader, header, tokenAuthCtx)

		t.Run("with valid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, validToken)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Equal(t, principal, usr)
			require.NoError(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Equal(t, extraWisdom, req.Context().Value(extra))
			assert.Nil(t, req.Context().Value(reason))
		})

		t.Run("with invalid token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, authPath, nil)
			require.NoError(t, err)
			req.Header.Set(apiKeyHeader, invalidToken)

			ok, usr, err := ta.Authenticate(req)
			assert.True(t, ok)
			assert.Nil(t, usr)
			require.Error(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Equal(t, expReason, req.Context().Value(reason))
			assert.Nil(t, req.Context().Value(extra))
		})

		t.Run("with missing token", func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?%s=%s", authPath, apiKeyParam, validToken), nil)
			require.NoError(t, err)

			ok, usr, err := ta.Authenticate(req)
			assert.False(t, ok)
			assert.Nil(t, usr)
			require.NoError(t, err)

			assert.Equal(t, wisdom, req.Context().Value(original))
			assert.Nil(t, req.Context().Value(reason))
			assert.Nil(t, req.Context().Value(extra))
		})
	})
}
