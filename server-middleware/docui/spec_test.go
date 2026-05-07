// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package docui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestServeSpecMiddleware(t *testing.T) {
	t.Run("ServeSpec handler", func(t *testing.T) {
		handler := ServeSpec(testSpec, nil)

		t.Run("serves spec", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger.json", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.EqualT(t, http.StatusOK, recorder.Code)

			responseHeaders := recorder.Result().Header
			responseContentType := responseHeaders.Get(contentTypeHeader)
			assert.EqualT(t, applicationJSON, responseContentType)

			require.JSONEqT(t, string(testSpec), recorder.Body.String())
		})

		t.Run("returns 404 when no next handler", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/pets", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.EqualT(t, http.StatusNotFound, recorder.Code)
		})

		t.Run("forwards to next handler for other url", func(t *testing.T) {
			handler = ServeSpec(testSpec, http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}))
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/pets", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.EqualT(t, http.StatusOK, recorder.Code)
		})
	})

	t.Run("ServeSpec handler with options", func(t *testing.T) {
		handler := ServeSpec(testSpec, nil,
			WithSpecPath("/swagger/spec/myapi-swagger.json"),
		)

		t.Run("serves spec", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger/spec/myapi-swagger.json", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.EqualT(t, http.StatusOK, recorder.Code)
		})

		t.Run("should not find spec there", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger.json", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.EqualT(t, http.StatusNotFound, recorder.Code)
		})
	})
}
