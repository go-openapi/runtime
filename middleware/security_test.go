// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	stdcontext "context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/runtime/internal/testing/petstore"
)

func TestSecurityMiddleware(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	context := NewContext(spec, api, nil)
	context.router = DefaultRouter(spec, context.api)
	mw := newSecureAPI(context, http.HandlerFunc(terminator))

	t.Run("without auth", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
		require.NoError(t, err)

		mw.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("with wrong password", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
		require.NoError(t, err)
		request.SetBasicAuth("admin", "wrong")

		mw.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		assert.NotEmpty(t, recorder.Header().Get("WWW-Authenticate"))
	})

	t.Run("with correct password", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
		require.NoError(t, err)
		request.SetBasicAuth("admin", "admin")

		mw.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("with unauthenticated path", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "//apipets/1", nil)
		require.NoError(t, err)

		mw.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
