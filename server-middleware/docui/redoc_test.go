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

func TestRedocMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		h := Redoc(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.EqualT(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))

		body := recorder.Body.String()
		assert.StringContainsT(t, body, fmt.Sprintf("<title>%s</title>", defaultDocsTitle))
		assert.StringContainsT(t, body, fmt.Sprintf("<redoc spec-url='%s'></redoc>", defaultDocsURL))
		assert.StringContainsT(t, body, redocLatest)
	})

	t.Run("with alternate path and spec URL", func(t *testing.T) {
		h := Redoc(nil,
			WithUIBasePath("/base"),
			WithUIPath("ui"),
			WithSpecURL("/ui/swagger.json"),
		)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/base/ui", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), "<redoc spec-url='/ui/swagger.json'></redoc>")
	})

	t.Run("with custom assets URL", func(t *testing.T) {
		h := Redoc(nil, WithUIAssetsURL("https://example.com/redoc.js"))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), `<script src="https://example.com/redoc.js">`)
	})

	t.Run("with custom template", func(t *testing.T) {
		const tpl = `<!DOCTYPE html>
<html>
  <body>
    <redoc spec-url='{{ .SpecURL }}' required-props-first=true></redoc>
    <script src="{{ .AssetsURL }}"> </script>
  </body>
</html>
`
		h := Redoc(nil, WithUITemplate(tpl))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), "required-props-first=true")
	})

	t.Run("falls through to next handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := Redoc(next)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusTeapot, recorder.Code)
	})

	t.Run("returns 404 when no next handler", func(t *testing.T) {
		h := Redoc(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with malformed template", func(t *testing.T) {
			assert.Panics(t, func() {
				Redoc(nil, WithUITemplate(malformedTemplate))
			})
		})

		t.Run("with template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				Redoc(nil, WithUITemplate(badTemplate))
			})
		})
	})
}

func TestUseRedoc(t *testing.T) {
	t.Run("composes as a func(http.Handler) http.Handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := UseRedoc()(next)

		t.Run("serves the docs page", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, req)
			assert.EqualT(t, http.StatusOK, recorder.Code)
		})

		t.Run("forwards everything else", func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, req)
			assert.EqualT(t, http.StatusTeapot, recorder.Code)
		})
	})
}
