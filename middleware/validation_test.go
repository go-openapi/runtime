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
	request.Header.Add("Accept", jsonMime)
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusBadRequest, recorder.Code)
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", jsonMime)
	request.Header.Add("Content-Type", "text/html")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusUnsupportedMediaType, recorder.Code)
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", strings.NewReader(`{"name":"dog"}`))
	request.Header.Add("Accept", jsonMime)
	request.Header.Add("Content-Type", "text/html")
	request.TransferEncoding = []string{"chunked"}

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, http.StatusUnsupportedMediaType, recorder.Code)
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", "application/json+special")
	request.Header.Add("Content-Type", "text/html")

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 406, recorder.Code)
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))

	// client sends data with unsupported mime
	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", jsonMime) // this content type is served by default by the API
	request.Header.Add("Content-Type", "application/json+special")
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 415, recorder.Code) // Unsupported media type
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))

	// client sends a body of data with no mime: breaks
	recorder = httptest.NewRecorder()
	request, _ = http.NewRequestWithContext(stdcontext.Background(), http.MethodPost, "/api/pets", nil)
	request.Header.Add("Accept", jsonMime)
	request.ContentLength = 1

	mw.ServeHTTP(recorder, request)
	assert.EqualT(t, 415, recorder.Code)
	assert.EqualT(t, jsonMime, recorder.Header().Get("Content-Type"))
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

// TestValidateContentType is a smoke test confirming the wrapper maps
// no-match to 415 (errors.InvalidContentType) and malformed-actual to 400
// (errors.NewParseError). The matching matrix lives in
// server-middleware/mediatype/match_test.go.
func TestValidateContentType(t *testing.T) {
	t.Run("nil allowed accepts anything", func(t *testing.T) {
		require.NoError(t, validateContentType(nil, jsonMime))
	})

	t.Run("match returns nil", func(t *testing.T) {
		require.NoError(t, validateContentType([]string{jsonMime}, jsonMime))
	})

	t.Run("no match returns 415", func(t *testing.T) {
		err := validateContentType([]string{jsonMime}, "text/html")
		require.Error(t, err)
		var v *errors.Validation
		require.ErrorAs(t, err, &v)
		assert.EqualT(t, http.StatusUnsupportedMediaType, int(v.Code()))
	})

	t.Run("malformed actual returns 400", func(t *testing.T) {
		// In the normal runtime flow this case is caught upstream by
		// runtime.ContentType. The smoke test exercises the direct path.
		err := validateContentType([]string{jsonMime}, "application(")
		require.Error(t, err)
		var p *errors.ParseError
		require.ErrorAs(t, err, &p)
		assert.EqualT(t, http.StatusBadRequest, int(p.Code()))
	})
}
