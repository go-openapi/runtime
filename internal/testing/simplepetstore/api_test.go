// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

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
	assert.JSONEq(t, swaggerJSON, rw.Body.String())
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
	assert.JSONEq(t, "[{\"id\":1,\"name\":\"Dog\",\"status\":\"available\"},{\"id\":2,\"name\":\"Cat\",\"status\":\"pending\"}]\n", rw.Body.String())
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
	assert.JSONEq(t, "{\"id\":1,\"name\":\"Dog\",\"status\":\"available\"}\n", rw.Body.String())
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
	assert.JSONEq(t, "{\"id\":3,\"name\":\"Fish\",\"status\":\"available\"}\n", rw.Body.String())
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
	assert.Empty(t, rw.Body.String())

	r, err = runtime.JSONRequest(http.MethodGet, "/api/pets/1", nil)
	require.NoError(t, err)
	r = r.WithContext(context.Background())
	rw = httptest.NewRecorder()
	handler.ServeHTTP(rw, r)
	assert.Equal(t, http.StatusNotFound, rw.Code)
	assert.JSONEq(t, "{\"code\":404,\"message\":\"not found: pet 1\"}", rw.Body.String())
}
