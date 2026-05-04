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

func TestValidateContentType(t *testing.T) {
	const (
		textPlain         = "text/plain"
		textPlainUTF8     = "text/plain;charset=utf-8"
		textPlainParamSrv = "text/plain; charset=utf-8"
	)
	data := []struct {
		hdr     string
		allowed []string
		err     *errors.Validation
	}{
		{"application/json", []string{"application/json"}, nil},
		{"application/json", []string{"application/x-yaml", "text/html"}, errors.InvalidContentType("application/json", []string{"application/x-yaml", "text/html"})},
		{"text/html; charset=utf-8", []string{"text/html"}, nil},
		{"text/html;charset=utf-8", []string{"text/html"}, nil},
		{"", []string{"application/json"}, errors.InvalidContentType("", []string{"application/json"})},
		{"text/html;           charset=utf-8", []string{"application/json"}, errors.InvalidContentType("text/html;           charset=utf-8", []string{"application/json"})},
		{"application(", []string{"application/json"}, errors.InvalidContentType("application(", []string{"application/json"})},
		{"application/json;char*", []string{"application/json"}, errors.InvalidContentType("application/json;char*", []string{"application/json"})},
		{"application/octet-stream", []string{"image/jpeg", "application/*"}, nil},
		{"image/png", []string{"*/*", "application/json"}, nil},
		// regression for https://github.com/go-openapi/runtime/issues/136:
		// allowed entries with MIME parameters should not block matching clients.
		// (1) client sends bare type, server allows type with params -> accept
		{textPlain, []string{textPlainParamSrv}, nil},
		// (2) client sends a different param than server -> reject
		{"text/plain;blah=true", []string{textPlainParamSrv},
			errors.InvalidContentType("text/plain;blah=true", []string{textPlainParamSrv})},
		// (3) client sends params, server allows bare type -> accept
		{textPlainUTF8, []string{textPlain}, nil},
		// (4) exact param match -> accept
		{textPlainUTF8, []string{textPlainUTF8}, nil},
		// param value compare is case-insensitive (charset is case-insensitive)
		{"text/plain;charset=UTF-8", []string{textPlainUTF8}, nil},
		// (5) conflicting param values -> reject
		{textPlainUTF8, []string{"text/plain;charset=ascii"},
			errors.InvalidContentType(textPlainUTF8, []string{"text/plain;charset=ascii"})},
	}

	for _, v := range data {
		err := validateContentType(v.allowed, v.hdr)
		if v.err == nil {
			require.NoError(t, err, "input: %q", v.hdr)
		} else {
			require.Error(t, err, "input: %q", v.hdr)
			assert.IsTypef(t, &errors.Validation{}, err, "input: %q", v.hdr)
			require.EqualErrorf(t, err, v.err.Error(), "input: %q", v.hdr)
			assert.EqualValues(t, http.StatusUnsupportedMediaType, err.(*errors.Validation).Code())
		}
	}
}
