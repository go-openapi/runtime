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

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationExecutor(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	api.RegisterOperation("get", "/pets", runtime.OperationHandlerFunc(func(_ interface{}) (interface{}, error) {
		return []interface{}{
			map[string]interface{}{"id": 1, "name": "a dog"},
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
	api.RegisterOperation("get", "/pets", runtime.OperationHandlerFunc(func(_ interface{}) (interface{}, error) {
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
