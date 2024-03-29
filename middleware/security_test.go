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

package middleware

import (
	stdcontext "context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
