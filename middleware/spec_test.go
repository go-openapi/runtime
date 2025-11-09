// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
)

func TestServeSpecMiddleware(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)

	t.Run("Spec handler", func(t *testing.T) {
		handler := Spec("", ctx.spec.Raw(), nil)

		t.Run("serves spec", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger.json", nil)
			require.NoError(t, err)
			request.Header.Add(runtime.HeaderContentType, runtime.JSONMime)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusOK, recorder.Code)

			responseHeaders := recorder.Result().Header
			responseContentType := responseHeaders.Get("Content-Type")
			assert.Equal(t, applicationJSON, responseContentType) //nolint:testifylint

			responseBody := recorder.Body
			require.NotNil(t, responseBody)
			require.JSONEq(t, string(spec.Raw()), responseBody.String())
		})

		t.Run("returns 404 when no next handler", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/pets", nil)
			require.NoError(t, err)
			request.Header.Add(runtime.HeaderContentType, runtime.JSONMime)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusNotFound, recorder.Code)
		})

		t.Run("forwards to next handler for other url", func(t *testing.T) {
			handler = Spec("", ctx.spec.Raw(), http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}))
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/pets", nil)
			require.NoError(t, err)
			request.Header.Add(runtime.HeaderContentType, runtime.JSONMime)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	})

	t.Run("Spec handler with options", func(t *testing.T) {
		handler := Spec("/swagger", ctx.spec.Raw(), nil,
			WithSpecPath("spec"),
			WithSpecDocument("myapi-swagger.json"),
		)

		t.Run("serves spec", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger/spec/myapi-swagger.json", nil)
			require.NoError(t, err)
			request.Header.Add(runtime.HeaderContentType, runtime.JSONMime)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusOK, recorder.Code)
		})

		t.Run("should not find spec there", func(t *testing.T) {
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/swagger.json", nil)
			require.NoError(t, err)
			request.Header.Add(runtime.HeaderContentType, runtime.JSONMime)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusNotFound, recorder.Code)
		})
	})
}
