// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	stdcontext "context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestOperationExecutor(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	api.RegisterOperation("get", "/pets", runtime.OperationHandlerFunc(func(_ any) (any, error) {
		return []any{
			map[string]any{"id": 1, "name": "a dog"},
		}, nil
	}))

	context := NewContext(spec, api, nil)
	context.router = DefaultRouter(spec, context.api)
	mw := NewOperationExecutor(context)

	recorder := httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request.Header.Add("Accept", "application/json")
	request.SetBasicAuth("admin", "admin")
	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `[{"id":1,"name":"a dog"}]`+"\n", recorder.Body.String())

	spec, api = petstore.NewAPI(t)
	api.RegisterOperation("get", "/pets", runtime.OperationHandlerFunc(func(_ any) (any, error) {
		return nil, errors.New(http.StatusUnprocessableEntity, "expected")
	}))

	context = NewContext(spec, api, nil)
	context.router = DefaultRouter(spec, context.api)
	mw = NewOperationExecutor(context)

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request.Header.Add("Accept", "application/json")
	request.SetBasicAuth("admin", "admin")
	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.JSONEq(t, `{"code":422,"message":"expected"}`, recorder.Body.String())
}
