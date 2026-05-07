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

func TestRapiDocMiddleware(t *testing.T) {
	t.Run("with defaults", func(t *testing.T) {
		h := RapiDoc(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.EqualT(t, "text/html; charset=utf-8", recorder.Header().Get(contentTypeHeader))

		body := recorder.Body.String()
		assert.StringContainsT(t, body, fmt.Sprintf("<title>%s</title>", defaultDocsTitle))
		assert.StringContainsT(t, body, fmt.Sprintf(`<rapi-doc spec-url="%s"></rapi-doc>`, defaultDocsURL))
		assert.StringContainsT(t, body, rapidocLatest)
	})

	t.Run("with alternate path and spec URL", func(t *testing.T) {
		h := RapiDoc(nil,
			WithUIBasePath("/base"),
			WithUIPath("ui"),
			WithSpecURL("/ui/swagger.json"),
		)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/base/ui", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), `<rapi-doc spec-url="/ui/swagger.json"></rapi-doc>`)
	})

	t.Run("with custom assets URL", func(t *testing.T) {
		h := RapiDoc(nil, WithUIAssetsURL("https://example.com/rapidoc.js"))
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
		assert.StringContainsT(t, recorder.Body.String(), `src="https://example.com/rapidoc.js"`)
	})

	t.Run("falls through to next handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := RapiDoc(next)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusTeapot, recorder.Code)
	})

	t.Run("returns 404 when no next handler", func(t *testing.T) {
		h := RapiDoc(nil)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/elsewhere", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("with template that fails to execute", func(t *testing.T) {
			assert.Panics(t, func() {
				RapiDoc(nil, WithUITemplate(badTemplate))
			})
		})
	})
}

func TestUseRapiDoc(t *testing.T) {
	t.Run("composes as a func(http.Handler) http.Handler", func(t *testing.T) {
		next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusTeapot)
		})
		h := UseRapiDoc()(next)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)
		assert.EqualT(t, http.StatusOK, recorder.Code)
	})
}
