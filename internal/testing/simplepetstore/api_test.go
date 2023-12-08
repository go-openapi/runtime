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

package simplepetstore

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimplePetstoreSpec(t *testing.T) {
	handler, err := NewPetstore()
	require.NoError(t, err)

	// Serves swagger spec document
	r, err := runtime.JSONRequest(http.MethodGet, "/swagger.json", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, swaggerJSON, rw.Body.String())
}

func TestSimplePetstoreAllPets(t *testing.T) {
	handler, err := NewPetstore()
	require.NoError(t, err)

	// Serves swagger spec document
	r, err := runtime.JSONRequest(http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "[{\"id\":1,\"name\":\"Dog\",\"status\":\"available\"},{\"id\":2,\"name\":\"Cat\",\"status\":\"pending\"}]\n", rw.Body.String())
}

func TestSimplePetstorePetByID(t *testing.T) {
	handler, err := NewPetstore()
	require.NoError(t, err)

	// Serves swagger spec document
	r, err := runtime.JSONRequest(http.MethodGet, "/api/pets/1", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "{\"id\":1,\"name\":\"Dog\",\"status\":\"available\"}\n", rw.Body.String())
}

func TestSimplePetstoreAddPet(t *testing.T) {
	handler, err := NewPetstore()
	require.NoError(t, err)

	// Serves swagger spec document
	r, err := runtime.JSONRequest(http.MethodPost, "/api/pets", bytes.NewBufferString(`{"name": "Fish","status": "available"}`))
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "{\"id\":3,\"name\":\"Fish\",\"status\":\"available\"}\n", rw.Body.String())
}

func TestSimplePetstoreDeletePet(t *testing.T) {
	handler, err := NewPetstore()
	require.NoError(t, err)

	// Serves swagger spec document
	r, err := runtime.JSONRequest(http.MethodDelete, "/api/pets/1", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, "", rw.Body.String())

	r, err = runtime.JSONRequest(http.MethodGet, "/api/pets/1", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw = httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusNotFound, rw.Code)
	assert.Equal(t, "{\"code\":404,\"message\":\"not found: pet 1\"}", rw.Body.String())
}
