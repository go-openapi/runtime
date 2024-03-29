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
	"sort"
	"strings"
	"testing"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func terminator(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func TestRouterMiddleware(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	context := NewContext(spec, api, nil)
	mw := NewRouter(context, http.HandlerFunc(terminator))

	recorder := httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodDelete, "/api/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)

	methods := strings.Split(recorder.Header().Get("Allow"), ",")
	sort.Strings(methods)
	assert.Equal(t, "GET,POST", strings.Join(methods, ","))

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/nopets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	spec, api = petstore.NewRootAPI(t)
	context = NewContext(spec, api, nil)
	mw = NewRouter(context, http.HandlerFunc(terminator))

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodDelete, "/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)

	methods = strings.Split(recorder.Header().Get("Allow"), ",")
	sort.Strings(methods)
	assert.Equal(t, "GET,POST", strings.Join(methods, ","))

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/nopets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestRouterBuilder(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	analyzed := analysis.New(spec.Spec())

	assert.Len(t, analyzed.RequiredConsumes(), 3)
	assert.Len(t, analyzed.RequiredProduces(), 5)
	assert.Len(t, analyzed.OperationIDs(), 4)

	// context := NewContext(spec, api)
	builder := petAPIRouterBuilder(spec, api, analyzed)
	getRecords := builder.records[http.MethodGet]
	postRecords := builder.records[http.MethodPost]
	deleteRecords := builder.records[http.MethodDelete]

	assert.Len(t, getRecords, 2)
	assert.Len(t, postRecords, 1)
	assert.Len(t, deleteRecords, 1)

	assert.Empty(t, builder.records[http.MethodPatch])
	assert.Empty(t, builder.records[http.MethodOptions])
	assert.Empty(t, builder.records[http.MethodHead])
	assert.Empty(t, builder.records[http.MethodPut])

	rec := postRecords[0]
	assert.Equal(t, "/pets", rec.Key)
	val := rec.Value.(*routeEntry)
	assert.Len(t, val.Consumers, 2)
	assert.Len(t, val.Producers, 2)
	assert.Len(t, val.Consumes, 2)
	assert.Len(t, val.Produces, 2)

	assert.Contains(t, val.Consumers, "application/json")
	assert.Contains(t, val.Producers, "application/x-yaml")
	assert.Contains(t, val.Consumes, "application/json")
	assert.Contains(t, val.Produces, "application/x-yaml")

	assert.Len(t, val.Parameters, 1)

	recG := getRecords[0]
	assert.Equal(t, "/pets", recG.Key)
	valG := recG.Value.(*routeEntry)
	assert.Len(t, valG.Consumers, 2)
	assert.Len(t, valG.Producers, 4)
	assert.Len(t, valG.Consumes, 2)
	assert.Len(t, valG.Produces, 4)

	assert.Len(t, valG.Parameters, 2)
}

func TestRouterCanonicalBasePath(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	spec.Spec().BasePath = "/api///"
	context := NewContext(spec, api, nil)
	mw := NewRouter(context, http.HandlerFunc(terminator))

	recorder := httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRouter_EscapedPath(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	spec.Spec().BasePath = "/api/"
	context := NewContext(spec, api, nil)
	mw := NewRouter(context, http.HandlerFunc(terminator))

	recorder := httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets/123", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets/abc%2Fdef", nil)
	require.NoError(t, err)

	mw.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
	ri, _, _ := context.RouteInfo(request)
	require.NotNil(t, ri)
	require.NotNil(t, ri.Params)
	assert.Equal(t, "abc/def", ri.Params.Get("id"))
}

func TestRouterStruct(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	router := DefaultRouter(spec, newRoutableUntypedAPI(spec, api, new(Context)))

	methods := router.OtherMethods("post", "/api/pets/{id}")
	assert.Len(t, methods, 2)

	entry, ok := router.Lookup("delete", "/api/pets/{id}")
	assert.True(t, ok)
	require.NotNil(t, entry)
	assert.Len(t, entry.Params, 1)
	assert.Equal(t, "id", entry.Params[0].Name)

	_, ok = router.Lookup("delete", "/pets")
	assert.False(t, ok)

	_, ok = router.Lookup("post", "/no-pets")
	assert.False(t, ok)
}

func petAPIRouterBuilder(spec *loads.Document, api *untyped.API, analyzed *analysis.Spec) *defaultRouteBuilder {
	builder := newDefaultRouteBuilder(spec, newRoutableUntypedAPI(spec, api, new(Context)))
	builder.AddRoute(http.MethodGet, "/pets", analyzed.AllPaths()["/pets"].Get)
	builder.AddRoute(http.MethodPost, "/pets", analyzed.AllPaths()["/pets"].Post)
	builder.AddRoute(http.MethodDelete, "/pets/{id}", analyzed.AllPaths()["/pets/{id}"].Delete)
	builder.AddRoute(http.MethodGet, "/pets/{id}", analyzed.AllPaths()["/pets/{id}"].Get)

	return builder
}

func TestPathConverter(t *testing.T) {
	cases := []struct {
		swagger string
		denco   string
	}{
		{"/", "/"},
		{"/something", "/something"},
		{"/{id}", "/:id"},
		{"/{id}/something/{anotherId}", "/:id/something/:anotherId"},
		{"/{petid}", "/:petid"},
		{"/{pet_id}", "/:pet_id"},
		{"/{petId}", "/:petId"},
		{"/{pet-id}", "/:pet-id"},
		// compost parameters tests
		{"/p_{pet_id}", "/p_:pet_id"},
		{"/p_{petId}.{petSubId}", "/p_:petId"},
	}

	for _, tc := range cases {
		actual := pathConverter.ReplaceAllString(tc.swagger, ":$1")
		assert.Equal(t, tc.denco, actual, "expected swagger path %s to match %s but got %s", tc.swagger, tc.denco, actual)
	}
}

func TestExtractCompositParameters(t *testing.T) {
	// name is the composite parameter's name, value is the value of this compost parameter, pattern is the pattern to be matched
	cases := []struct {
		name    string
		value   string
		pattern string
		names   []string
		values  []string
	}{
		{name: "fragment", value: "gie", pattern: "e", names: []string{"fragment"}, values: []string{"gi"}},
		{name: "fragment", value: "t.simpson", pattern: ".{subfragment}", names: []string{"fragment", "subfragment"}, values: []string{"t", "simpson"}},
	}
	for _, tc := range cases {
		names, values := decodeCompositParams(tc.name, tc.value, tc.pattern, nil, nil)
		assert.EqualValues(t, tc.names, names)
		assert.EqualValues(t, tc.values, values)
	}
}
