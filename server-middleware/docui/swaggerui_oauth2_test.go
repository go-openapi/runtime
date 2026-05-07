// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package docui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestSwaggerUIOAuth2CallbackMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		h := SwaggerUIOAuth2Callback(nil)

		// Default callback URL is /<basepath>/<path>/oauth2-callback,
		// i.e. /docs/oauth2-callback.
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/oauth2-callback", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		require.EqualT(t, http.StatusOK, recorder.Code)
		assert.EqualT(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))

		body := recorder.Body.String()
		assert.StringContainsT(t, body, fmt.Sprintf("<title>%s</title>", defaultDocsTitle))
		// Marker from the swagger-ui-dist OAuth2 popup callback script.
		assert.StringContainsT(t, body, `oauth2.auth.schema.get("flow") === "accessCode"`)
	})

	t.Run("with explicit OAuth2CallbackURL", func(t *testing.T) {
		h := SwaggerUIOAuth2Callback(nil, WithSwaggerUIOptions(SwaggerUIOptions{
			OAuth2CallbackURL: "/custom/callback",
		}))

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/custom/callback", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})

	t.Run("with alternate base path and path", func(t *testing.T) {
		h := SwaggerUIOAuth2Callback(nil,
			WithUIBasePath("/api"),
			WithUIPath("ui"),
		)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/ui/oauth2-callback", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 404 when no next handler", func(t *testing.T) {
		h := SwaggerUIOAuth2Callback(nil)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/nowhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("falls through to next handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := SwaggerUIOAuth2Callback(next)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusTeapot, recorder.Code)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				SwaggerUIOAuth2Callback(nil, WithUITemplate(badTemplate))
			})
		})
	})
}

func TestUseSwaggerUIOAuth2Callback(t *testing.T) {
	t.Run("composes as a func(http.Handler) http.Handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := UseSwaggerUIOAuth2Callback()(next)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/oauth2-callback", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})
}
