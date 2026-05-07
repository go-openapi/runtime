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

func TestSwaggerUIMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		h := SwaggerUI(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.EqualT(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))

		body := recorder.Body.String()
		assert.StringContainsT(t, body, fmt.Sprintf("<title>%s</title>", defaultDocsTitle))
		// html/template JS-escapes '/' as '\/' inside <script> context.
		assert.StringContainsT(t, body, `url: '\/swagger.json',`)
		assert.StringContainsT(t, body, swaggerLatest)
		// SwaggerUI-specific defaults are filled in swaggeruiSetup after
		// user opts apply, so they must show up unconditionally.
		assert.StringContainsT(t, body, swaggerPresetLatest)
		assert.StringContainsT(t, body, swaggerStylesLatest)
		assert.StringContainsT(t, body, swaggerFavicon16Latest)
		assert.StringContainsT(t, body, swaggerFavicon32Latest)
	})

	t.Run("with trailing slash on path (issue #238)", func(t *testing.T) {
		h := SwaggerUI(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs/", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns 404 when no next handler", func(t *testing.T) {
		h := SwaggerUI(nil)
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
		h := SwaggerUI(next)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusTeapot, recorder.Code)
	})

	t.Run("with alternate path and spec URL", func(t *testing.T) {
		h := SwaggerUI(nil,
			WithUIBasePath("/base"),
			WithUIPath("ui"),
			WithSpecURL("/ui/swagger.json"),
		)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/base/ui", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), `url: '\/ui\/swagger.json',`)
	})

	t.Run("WithSwaggerUIOptions does not clobber filled-in defaults", func(t *testing.T) {
		// Empty struct should not erase the SwaggerUI defaults — they're
		// applied AFTER user opts in swaggeruiSetup.
		h := SwaggerUI(nil, WithSwaggerUIOptions(SwaggerUIOptions{}))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		assert.StringContainsT(t, body, swaggerPresetLatest)
		assert.StringContainsT(t, body, swaggerStylesLatest)
		assert.StringContainsT(t, body, swaggerFavicon16Latest)
		assert.StringContainsT(t, body, swaggerFavicon32Latest)
	})

	t.Run("with custom SwaggerUI fields preserved", func(t *testing.T) {
		h := SwaggerUI(nil, WithSwaggerUIOptions(SwaggerUIOptions{
			SwaggerPresetURL: "https://example.com/preset.js",
			SwaggerStylesURL: "https://example.com/style.css",
		}))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		body := recorder.Body.String()
		assert.StringContainsT(t, body, "https://example.com/preset.js")
		assert.StringContainsT(t, body, "https://example.com/style.css")
		// favicons were not provided; defaults apply.
		assert.StringContainsT(t, body, swaggerFavicon16Latest)
	})

	t.Run("OAuth2CallbackURL appears in the JS scriptlet when set", func(t *testing.T) {
		h := SwaggerUI(nil, WithSwaggerUIOptions(SwaggerUIOptions{
			OAuth2CallbackURL: "/docs/oauth2-callback",
		}))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.StringContainsT(t, recorder.Body.String(), `oauth2RedirectUrl: '\/docs\/oauth2-callback'`)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				SwaggerUI(nil, WithUITemplate(badTemplate))
			})
		})
	})
}

func TestUseSwaggerUI(t *testing.T) {
	t.Run("composes as a func(http.Handler) http.Handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := UseSwaggerUI()(next)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})
}
