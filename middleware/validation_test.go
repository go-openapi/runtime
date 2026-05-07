// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"bytes"
	stdcontext "context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func newTestValidation(ctx *Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		matched, rCtx, _ := ctx.RouteInfo(r)
		if rCtx != nil {
			r = rCtx
		}
		if matched == nil {
			ctx.NotFound(rw, r)
			return
		}
		_, r, result := ctx.BindAndValidate(r, matched)

		if result != nil {
			ctx.Respond(rw, r, matched.Produces, matched, result)
			return
		}

		next.ServeHTTP(rw, r)
	})
}

func TestContentTypeValidation(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	context := NewContext(spec, api, nil)
	context.router = DefaultRouter(spec, context.api)

	mw := newTestValidation(context, http.HandlerFunc(terminator))

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodGet, "/api/pets", nil)
	request.Header.Add("Accept", "*/*")
	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Content-Type", "application(")
	request.Header.Add("Accept", "application/json")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusBadRequest, recorder.Code)
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "text/html")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusUnsupportedMediaType, recorder.Code)
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", strings.NewReader(`{"name":"dog"}`))
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "text/html")
	request.TransferEncoding = []string{"chunked"}

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusUnsupportedMediaType, recorder.Code)
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "application/json+special")
	request.Header.Add("Content-Type", "text/html")

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 406, recorder.Code)
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))

	// client sends data with unsupported mime
	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "application/json") // this content type is served by default by the API
	request.Header.Add("Content-Type", "application/json+special")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 415, recorder.Code) // Unsupported media type
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))

	// client sends a body of data with no mime: breaks
	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "application/json")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 415, recorder.Code)
	assert.EqualT(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestResponseFormatValidation(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	context := NewContext(spec, api, nil)
	context.router = DefaultRouter(spec, context.api)
	mw := newTestValidation(context, http.HandlerFunc(terminator))

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", bytes.NewBufferString(`name: Dog`))
	request.Header.Set(runtime.HeaderContentType, "application/x-yaml")
	request.Header.Set(runtime.HeaderAccept, "application/x-yaml")

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 200, recorder.Code, recorder.Body.String())

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", bytes.NewBufferString(`name: Dog`))
	request.Header.Set(runtime.HeaderContentType, "application/x-yaml")
	request.Header.Set(runtime.HeaderAccept, "application/sml")

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusNotAcceptable, recorder.Code)
}

// TestValidateContentType is a smoke test confirming the wrapper still
// maps to errors.InvalidContentType for both no-match and malformed-actual.
// The matching matrix lives in server-middleware/mediatype/match_test.go.
func TestValidateContentType(t *testing.T) {
	const json = "application/json"

	t.Run("nil allowed accepts anything", func(t *testing.T) {
		require.NoError(t, validateContentType(nil, json))
	})

	t.Run("match returns nil", func(t *testing.T) {
		require.NoError(t, validateContentType([]string{json}, json))
	})

	t.Run("no match returns 415", func(t *testing.T) {
		err := validateContentType([]string{json}, "text/html")
		require.Error(t, err)
		var v *errors.Validation
		require.ErrorAs(t, err, &v)
		assert.EqualT(t, http.StatusUnsupportedMediaType, int(v.Code()))
	})

	t.Run("malformed actual maps to the same 415 today", func(t *testing.T) {
		// Behaviour preservation: callers wanting a 400-vs-415 split can
		// inspect mediatype.MatchFirst's error directly.
		err := validateContentType([]string{json}, "application(")
		require.Error(t, err)
		var v *errors.Validation
		require.ErrorAs(t, err, &v)
		assert.EqualT(t, http.StatusUnsupportedMediaType, int(v.Code()))
	})
}
