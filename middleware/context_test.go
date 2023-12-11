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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/loads/fmts"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/runtime/middleware/untyped"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBindRequester struct {
}

func (s *stubBindRequester) BindRequest(*http.Request, *MatchedRoute) error {
	return nil
}

type stubOperationHandler struct {
}

func (s *stubOperationHandler) ParameterModel() interface{} {
	return nil
}

func (s *stubOperationHandler) Handle(_ interface{}) (interface{}, error) {
	return map[string]interface{}{}, nil
}

func init() {
	loads.AddLoader(fmts.YAMLMatcher, fmts.YAMLDoc)
}

func assertAPIError(t *testing.T, wantCode int, err error) {
	t.Helper()

	require.Error(t, err)

	ce, ok := err.(*apierrors.CompositeError)
	assert.True(t, ok)
	assert.NotEmpty(t, ce.Errors)

	ae, ok := ce.Errors[0].(apierrors.Error)
	assert.True(t, ok)
	assert.Equal(t, wantCode, int(ae.Code()))
}

func TestContentType_Issue264(t *testing.T) {
	swspec, err := loads.Spec("../fixtures/bugs/264/swagger.yml")
	require.NoError(t, err)

	api := untyped.NewAPI(swspec)
	api.RegisterConsumer(applicationJSON, runtime.JSONConsumer())
	api.RegisterProducer(applicationJSON, runtime.JSONProducer())
	api.RegisterOperation("delete", "/key/{id}", new(stubOperationHandler))

	handler := Serve(swspec, api)
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodDelete, "/key/1", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestContentType_Issue172(t *testing.T) {
	swspec, err := loads.Spec("../fixtures/bugs/172/swagger.yml")
	require.NoError(t, err)

	api := untyped.NewAPI(swspec)
	api.RegisterConsumer("application/vnd.cia.v1+json", runtime.JSONConsumer())
	api.RegisterProducer("application/vnd.cia.v1+json", runtime.JSONProducer())
	api.RegisterOperation("get", "/pets", new(stubOperationHandler))

	handler := Serve(swspec, api)
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/pets", nil)
	require.NoError(t, err)

	request.Header.Add("Accept", "application/json+special")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotAcceptable, recorder.Code)

	// acceptable as defined as default by the API (not explicit in the spec)
	request.Header.Add("Accept", applicationJSON)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestContentType_Issue174(t *testing.T) {
	swspec, err := loads.Spec("../fixtures/bugs/174/swagger.yml")
	require.NoError(t, err)

	api := untyped.NewAPI(swspec)
	api.RegisterConsumer(applicationJSON, runtime.JSONConsumer())
	api.RegisterProducer(applicationJSON, runtime.JSONProducer())
	api.RegisterOperation("get", "/pets", new(stubOperationHandler))

	handler := Serve(swspec, api)
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/pets", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

const (
	testHost = "https://localhost:8080"

	// how to get the spec document?
	defaultSpecPath = "/swagger.json"
	defaultSpecURL  = testHost + defaultSpecPath
	// how to get the UI asset?
	defaultUIURL = testHost + "/api/docs"
)

func TestServe(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	handler := Serve(spec, api)

	t.Run("serve spec document", func(t *testing.T) {
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultSpecURL, nil)
		require.NoError(t, err)

		request.Header.Add("Content-Type", runtime.JSONMime)
		request.Header.Add("Accept", runtime.JSONMime)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should not find UI there", func(t *testing.T) {
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, testHost+"/swagger-ui", nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("should find UI here", func(t *testing.T) {
		request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultUIURL, nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusOK, recorder.Code)

		htmlResponse := recorder.Body.String()
		assert.Containsf(t, htmlResponse, "<title>Swagger Petstore</title>", "should default to the API's title")
		assert.Containsf(t, htmlResponse, "<redoc", "should default to Redoc UI")
		assert.Containsf(t, htmlResponse, "spec-url='/swagger.json'>", "should default to /swagger.json spec document")
	})
}

func TestServeWithUIs(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)

	const (
		alternateSpecURL  = testHost + "/specs/petstore.json"
		alternateSpecPath = "/specs/petstore.json"
		alternateUIURL    = testHost + "/ui/docs"
	)

	uiOpts := []UIOption{
		WithUIBasePath("ui"), // override the base path from the spec, implies /ui
		WithUIPath("docs"),
		WithUISpecURL("/specs/petstore.json"),
	}

	t.Run("with APIHandler", func(t *testing.T) {
		t.Run("with defaults", func(t *testing.T) {
			handler := ctx.APIHandler(nil)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Containsf(t, htmlResponse, "<redoc", "should default to Redoc UI")
			})

			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})

		t.Run("with options", func(t *testing.T) {
			handler := ctx.APIHandler(nil, uiOpts...)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Contains(t, htmlResponse, fmt.Sprintf("<redoc spec-url='%s'></redoc>", alternateSpecPath))
			})

			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})
	})

	t.Run("with APIHandlerSwaggerUI", func(t *testing.T) {
		t.Run("with defaults", func(t *testing.T) {
			handler := ctx.APIHandlerSwaggerUI(nil)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Contains(t, htmlResponse, fmt.Sprintf(`url: '%s',`, strings.ReplaceAll(defaultSpecPath, `/`, `\/`)))
			})

			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})

		t.Run("with options", func(t *testing.T) {
			handler := ctx.APIHandlerSwaggerUI(nil, uiOpts...)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Contains(t, htmlResponse, fmt.Sprintf(`url: '%s',`, strings.ReplaceAll(alternateSpecPath, `/`, `\/`)))
			})

			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})
	})

	t.Run("with APIHandlerRapiDoc", func(t *testing.T) {
		t.Run("with defaults", func(t *testing.T) {
			handler := ctx.APIHandlerRapiDoc(nil)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Contains(t, htmlResponse, fmt.Sprintf("<rapi-doc spec-url=%q></rapi-doc>", defaultSpecPath))
			})

			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, defaultSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})

		t.Run("with options", func(t *testing.T) {
			handler := ctx.APIHandlerRapiDoc(nil, uiOpts...)

			t.Run("should find UI", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateUIURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)

				htmlResponse := recorder.Body.String()
				assert.Contains(t, htmlResponse, fmt.Sprintf("<rapi-doc spec-url=%q></rapi-doc>", alternateSpecPath))
			})
			t.Run("should find spec", func(t *testing.T) {
				request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, alternateSpecURL, nil)
				require.NoError(t, err)
				recorder := httptest.NewRecorder()

				handler.ServeHTTP(recorder, request)
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		})
	})
}

func TestContextAuthorize(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := runtime.JSONRequest(http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request = request.WithContext(stdcontext.Background())

	ri, reqWithCtx, ok := ctx.RouteInfo(request)
	assert.True(t, ok)
	require.NotNil(t, reqWithCtx)

	request = reqWithCtx

	p, reqWithCtx, err := ctx.Authorize(request, ri)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Nil(t, reqWithCtx)

	v := request.Context().Value(ctxSecurityPrincipal)
	assert.Nil(t, v)

	request.SetBasicAuth("wrong", "wrong")
	p, reqWithCtx, err = ctx.Authorize(request, ri)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Nil(t, reqWithCtx)

	v = request.Context().Value(ctxSecurityPrincipal)
	assert.Nil(t, v)

	request.SetBasicAuth("admin", "admin")
	p, reqWithCtx, err = ctx.Authorize(request, ri)
	require.NoError(t, err)
	assert.Equal(t, "admin", p)
	require.NotNil(t, reqWithCtx)

	// Assign the new returned request to follow with the test
	request = reqWithCtx

	v, ok = request.Context().Value(ctxSecurityPrincipal).(string)
	assert.True(t, ok)
	assert.Equal(t, "admin", v)

	// Once the request context contains the principal the authentication
	// isn't rechecked
	request.SetBasicAuth("doesn't matter", "doesn't")
	pp, reqCtx, rr := ctx.Authorize(request, ri)
	assert.Equal(t, p, pp)
	assert.Equal(t, err, rr)
	assert.Equal(t, request, reqCtx)
}

func TestContextAuthorize_WithAuthorizer(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := runtime.JSONRequest(http.MethodPost, "/api/pets", nil)
	require.NoError(t, err)
	request = request.WithContext(stdcontext.Background())

	ri, reqWithCtx, ok := ctx.RouteInfo(request)
	assert.True(t, ok)
	require.NotNil(t, reqWithCtx)

	request = reqWithCtx

	request.SetBasicAuth("topuser", "topuser")
	p, reqWithCtx, err := ctx.Authorize(request, ri)
	assertAPIError(t, apierrors.InvalidTypeCode, err)
	assert.Nil(t, p)
	assert.Nil(t, reqWithCtx)

	request.SetBasicAuth("admin", "admin")
	p, reqWithCtx, err = ctx.Authorize(request, ri)
	require.NoError(t, err)
	assert.Equal(t, "admin", p)
	require.NotNil(t, reqWithCtx)

	request.SetBasicAuth("anyother", "anyother")
	p, reqWithCtx, err = ctx.Authorize(request, ri)
	require.Error(t, err)
	ae, ok := err.(apierrors.Error)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, int(ae.Code()))
	assert.Nil(t, p)
	assert.Nil(t, reqWithCtx)
}

func TestContextNegotiateContentType(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	// request.Header.Add("Accept", "*/*")
	request.Header.Add("content-type", "text/html")

	v := request.Context().Value(ctxBoundParams)
	assert.Nil(t, v)

	ri, request, _ := ctx.RouteInfo(request)

	res := NegotiateContentType(request, ri.Produces, "text/plain")
	assert.Equal(t, ri.Produces[0], res)
}

func TestContextBindValidRequest(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	// invalid content-type value
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", strings.NewReader(`{"name":"dog"}`))
	require.NoError(t, err)
	request.Header.Add("content-type", "/json")

	ri, request, _ := ctx.RouteInfo(request)
	assertAPIError(t, 400, ctx.BindValidRequest(request, ri, new(stubBindRequester)))

	// unsupported content-type value
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", strings.NewReader(`{"name":"dog"}`))
	require.NoError(t, err)
	request.Header.Add("content-type", "text/html")

	ri, request, _ = ctx.RouteInfo(request)
	assertAPIError(t, http.StatusUnsupportedMediaType, ctx.BindValidRequest(request, ri, new(stubBindRequester)))

	// unacceptable accept value
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	require.NoError(t, err)
	request.Header.Add("Accept", "application/vnd.cia.v1+json")
	request.Header.Add("content-type", applicationJSON)

	ri, request, _ = ctx.RouteInfo(request)
	assertAPIError(t, http.StatusNotAcceptable, ctx.BindValidRequest(request, ri, new(stubBindRequester)))
}

func TestContextBindValidRequest_Issue174(t *testing.T) {
	spec, err := loads.Spec("../fixtures/bugs/174/swagger.yml")
	require.NoError(t, err)

	api := untyped.NewAPI(spec)
	api.RegisterConsumer(applicationJSON, runtime.JSONConsumer())
	api.RegisterProducer(applicationJSON, runtime.JSONProducer())
	api.RegisterOperation("get", "/pets", new(stubOperationHandler))

	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/pets", nil)
	ri, request, _ := ctx.RouteInfo(request)
	require.NoError(t, ctx.BindValidRequest(request, ri, new(stubBindRequester)))
}

func TestContextBindAndValidate(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "*/*")
	request.Header.Add("content-type", "text/html")
	request.ContentLength = 1

	v := request.Context().Value(ctxBoundParams)
	assert.Nil(t, v)

	ri, request, _ := ctx.RouteInfo(request)
	data, request, result := ctx.BindAndValidate(request, ri) // this requires a much more thorough test
	assert.NotNil(t, data)
	require.Error(t, result)

	v, ok := request.Context().Value(ctxBoundParams).(*validation)
	assert.True(t, ok)
	assert.NotNil(t, v)

	dd, rCtx, rr := ctx.BindAndValidate(request, ri)
	assert.Equal(t, data, dd)
	assert.Equal(t, result, rr)
	assert.Equal(t, rCtx, request)
}

func TestContextRender(t *testing.T) {
	ct := runtime.JSONMime
	spec, api := petstore.NewAPI(t)
	assert.NotNil(t, spec)
	assert.NotNil(t, api)

	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	request.Header.Set(runtime.HeaderAccept, ct)
	ri, request, _ := ctx.RouteInfo(request)

	recorder := httptest.NewRecorder()
	ctx.Respond(recorder, request, []string{ct}, ri, map[string]interface{}{"name": "hello"})
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "{\"name\":\"hello\"}\n", recorder.Body.String())

	recorder = httptest.NewRecorder()
	ctx.Respond(recorder, request, []string{ct}, ri, errors.New("this went wrong"))
	assert.Equal(t, 500, recorder.Code)

	// recorder = httptest.NewRecorder()
	// assert.Panics(t, func() { ctx.Respond(recorder, request, []string{ct}, ri, map[int]interface{}{1: "hello"}) })

	// Panic when route is nil and there is not a producer for the requested response format
	recorder = httptest.NewRecorder()
	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderAccept, "text/xml")
	assert.Panics(t, func() { ctx.Respond(recorder, request, []string{}, nil, map[string]interface{}{"name": "hello"}) })

	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderAccept, ct)
	ri, request, _ = ctx.RouteInfo(request)

	recorder = httptest.NewRecorder()
	ctx.Respond(recorder, request, []string{ct}, ri, map[string]interface{}{"name": "hello"})
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "{\"name\":\"hello\"}\n", recorder.Body.String())

	recorder = httptest.NewRecorder()
	ctx.Respond(recorder, request, []string{ct}, ri, errors.New("this went wrong"))
	assert.Equal(t, 500, recorder.Code)

	// recorder = httptest.NewRecorder()
	// assert.Panics(t, func() { ctx.Respond(recorder, request, []string{ct}, ri, map[int]interface{}{1: "hello"}) })

	// recorder = httptest.NewRecorder()
	// request, _ = http.NewRequestWithContext(stdcontext.Background(),http.MethodGet, "/pets", nil)
	// assert.Panics(t, func() { ctx.Respond(recorder, request, []string{}, ri, map[string]interface{}{"name": "hello"}) })

	recorder = httptest.NewRecorder()
	request, err = http.NewRequestWithContext(stdcontext.Background(), http.MethodDelete, "/api/pets/1", nil)
	require.NoError(t, err)
	ri, request, _ = ctx.RouteInfo(request)
	ctx.Respond(recorder, request, ri.Produces, ri, nil)
	assert.Equal(t, 204, recorder.Code)
}

func TestContextValidResponseFormat(t *testing.T) {
	const ct = applicationJSON
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "http://localhost:8080", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderAccept, ct)

	// check there's nothing there
	cached, ok := request.Context().Value(ctxResponseFormat).(string)
	assert.False(t, ok)
	assert.Empty(t, cached)

	// trigger the parse
	mt, request := ctx.ResponseFormat(request, []string{ct})
	assert.Equal(t, ct, mt)

	// check it was cached
	cached, ok = request.Context().Value(ctxResponseFormat).(string)
	assert.True(t, ok)
	assert.Equal(t, ct, cached)

	// check if the cast works and fetch from cache too
	mt, _ = ctx.ResponseFormat(request, []string{ct})
	assert.Equal(t, ct, mt)
}

func TestContextInvalidResponseFormat(t *testing.T) {
	ct := "application/x-yaml"
	other := "application/sgml"
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "http://localhost:8080", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderAccept, ct)

	// check there's nothing there
	cached, ok := request.Context().Value(ctxResponseFormat).(string)
	assert.False(t, ok)
	assert.Empty(t, cached)

	// trigger the parse
	mt, request := ctx.ResponseFormat(request, []string{other})
	assert.Empty(t, mt)

	// check it was cached
	cached, ok = request.Context().Value(ctxResponseFormat).(string)
	assert.False(t, ok)
	assert.Empty(t, cached)

	// check if the cast works and fetch from cache too
	mt, rCtx := ctx.ResponseFormat(request, []string{other})
	assert.Empty(t, mt)
	assert.Equal(t, request, rCtx)
}

func TestContextValidRoute(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)

	// check there's nothing there
	cached := request.Context().Value(ctxMatchedRoute)
	assert.Nil(t, cached)

	matched, rCtx, ok := ctx.RouteInfo(request)
	assert.True(t, ok)
	assert.NotNil(t, matched)
	assert.NotNil(t, rCtx)
	assert.NotEqual(t, request, rCtx)

	request = rCtx

	// check it was cached
	_, ok = request.Context().Value(ctxMatchedRoute).(*MatchedRoute)
	assert.True(t, ok)

	matched, rCtx, ok = ctx.RouteInfo(request)
	assert.True(t, ok)
	assert.NotNil(t, matched)
	assert.Equal(t, request, rCtx)
}

func TestContextInvalidRoute(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodDelete, "pets", nil)
	require.NoError(t, err)

	// check there's nothing there
	cached := request.Context().Value(ctxMatchedRoute)
	assert.Nil(t, cached)

	matched, rCtx, ok := ctx.RouteInfo(request)
	assert.False(t, ok)
	assert.Nil(t, matched)
	assert.Nil(t, rCtx)

	// check it was not cached
	cached = request.Context().Value(ctxMatchedRoute)
	assert.Nil(t, cached)

	matched, rCtx, ok = ctx.RouteInfo(request)
	assert.False(t, ok)
	assert.Nil(t, matched)
	assert.Nil(t, rCtx)
}

func TestContextValidContentType(t *testing.T) {
	ct := applicationJSON
	ctx := NewContext(nil, nil, nil)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "http://localhost:8080", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderContentType, ct)

	// check there's nothing there
	cached := request.Context().Value(ctxContentType)
	assert.Nil(t, cached)

	// trigger the parse
	mt, _, rCtx, err := ctx.ContentType(request)
	require.NoError(t, err)
	assert.Equal(t, ct, mt)
	assert.NotNil(t, rCtx)
	assert.NotEqual(t, request, rCtx)

	request = rCtx

	// check it was cached
	cached = request.Context().Value(ctxContentType)
	assert.NotNil(t, cached)

	// check if the cast works and fetch from cache too
	mt, _, rCtx, err = ctx.ContentType(request)
	require.NoError(t, err)
	assert.Equal(t, ct, mt)
	assert.Equal(t, request, rCtx)
}

func TestContextInvalidContentType(t *testing.T) {
	ct := "application("
	ctx := NewContext(nil, nil, nil)

	request, err := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "http://localhost:8080", nil)
	require.NoError(t, err)
	request.Header.Set(runtime.HeaderContentType, ct)

	// check there's nothing there
	cached := request.Context().Value(ctxContentType)
	assert.Nil(t, cached)

	// trigger the parse
	mt, _, rCtx, err := ctx.ContentType(request)
	require.Error(t, err)
	assert.Empty(t, mt)
	assert.Nil(t, rCtx)

	// check it was not cached
	cached = request.Context().Value(ctxContentType)
	assert.Nil(t, cached)

	// check if the failure continues
	_, _, rCtx, err = ctx.ContentType(request)
	require.Error(t, err)
	assert.Nil(t, rCtx)
}
