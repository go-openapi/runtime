// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware_test

// Smoke tests for the deprecated middleware aliases that forward to the
// docui and negotiate packages. These verify that:
//
//   - the type aliases still resolve so user code keeps compiling,
//   - the function-value aliases still produce the documented behaviour.
//
// The exhaustive coverage lives in the destination packages themselves.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/server-middleware/docui"
	"github.com/go-openapi/runtime/server-middleware/negotiate"
)

// Compile-time assertions that the deprecated middleware names alias the
// destination types — type identity is required for these assignments to
// type-check.
var (
	_ = func(o docui.SwaggerUIOpts) middleware.SwaggerUIOpts { return o }
	_ = func(o docui.RedocOpts) middleware.RedocOpts { return o }
	_ = func(o docui.RapiDocOpts) middleware.RapiDocOpts { return o }
	_ = func(o docui.UIOption) middleware.UIOption { return o }
	_ = func(o docui.SpecOption) middleware.SpecOption { return o }
	_ = func(o negotiate.Option) middleware.NegotiateOption { return o }
)

func TestDeprecatedDocUIForwarders(t *testing.T) {
	t.Run("middleware.SwaggerUI still serves the docs page", func(t *testing.T) {
		h := middleware.SwaggerUI(middleware.SwaggerUIOpts{}, nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.EqualT(t, http.StatusOK, rec.Code)
	})

	t.Run("middleware.Redoc still serves the docs page", func(t *testing.T) {
		h := middleware.Redoc(middleware.RedocOpts{}, nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.EqualT(t, http.StatusOK, rec.Code)
	})

	t.Run("middleware.RapiDoc still serves the docs page", func(t *testing.T) {
		h := middleware.RapiDoc(middleware.RapiDocOpts{}, nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.EqualT(t, http.StatusOK, rec.Code)
	})

	t.Run("middleware.SwaggerUIOAuth2Callback still serves the callback page", func(t *testing.T) {
		h := middleware.SwaggerUIOAuth2Callback(middleware.SwaggerUIOpts{}, nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/oauth2-callback", nil)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.EqualT(t, http.StatusOK, rec.Code)
	})

	t.Run("middleware.Spec still serves the spec document", func(t *testing.T) {
		body := []byte(`{"swagger":"2.0"}`)
		h := middleware.Spec("", body, nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger.json", nil)
		require.NoError(t, err)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.EqualT(t, http.StatusOK, rec.Code)
		assert.EqualT(t, string(body), rec.Body.String())
	})

	t.Run("middleware.NegotiateContentType still selects the offered type", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/json")
		got := middleware.NegotiateContentType(req, []string{"application/json"}, "")
		assert.EqualT(t, "application/json", got)
	})

	t.Run("middleware.NegotiateContentEncoding still selects gzip", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		got := middleware.NegotiateContentEncoding(req, []string{"identity", "gzip"})
		assert.EqualT(t, "gzip", got)
	})

	t.Run("middleware.WithIgnoreParameters still strips params", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "text/plain;charset=ascii")
		// Strict (default): no match because charset disagrees.
		strict := middleware.NegotiateContentType(req, []string{"text/plain;charset=utf-8"}, "fallback")
		assert.EqualT(t, "fallback", strict)
		// Loose (legacy mode): bare types agree, offer picked.
		loose := middleware.NegotiateContentType(req, []string{"text/plain;charset=utf-8"}, "fallback",
			middleware.WithIgnoreParameters(true),
		)
		assert.EqualT(t, "text/plain;charset=utf-8", loose)
	})
}
