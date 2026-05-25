// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build openapi_unsafe_skipauth

package middleware

import (
	stdcontext "context"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestContextAuthorize_SkipAuth covers the dev-only bypass enabled by
// the `openapi_unsafe_skipauth` build tag. With SetSkipAuth(true) any
// request to a secured route resolves to (nil principal, original
// request, nil error). Reset SetSkipAuth(false) at the end so other
// tagged tests in the same binary observe the default behavior.
func TestContextAuthorize_SkipAuth(t *testing.T) {
	t.Cleanup(func() { SetSkipAuth(false) })

	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	request, err := runtime.JSONRequest(http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)
	request = request.WithContext(stdcontext.Background())

	ri, reqWithCtx, ok := ctx.RouteInfo(request)
	require.True(t, ok)
	require.NotNil(t, reqWithCtx)
	request = reqWithCtx

	// Baseline: without skip, the unsecured request is rejected.
	p, reqOut, err := ctx.Authorize(request, ri)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Nil(t, reqOut)

	// Enable the bypass: same request now succeeds with nil principal.
	SetSkipAuth(true)
	p, reqOut, err = ctx.Authorize(request, ri)
	require.NoError(t, err)
	assert.Nil(t, p)
	assert.Equal(t, request, reqOut, "request must be returned unchanged when bypassed")

	// Disabling the bypass restores rejection.
	SetSkipAuth(false)
	p, reqOut, err = ctx.Authorize(request, ri)
	require.Error(t, err)
	assert.Nil(t, p)
	assert.Nil(t, reqOut)
}

// TestContextAuthorize_SkipAuth_NilRoute confirms the bypass path
// also covers the pre-existing nil-route fast path: Authorize must
// not panic and returns the same (nil, nil, nil) shape as production.
func TestContextAuthorize_SkipAuth_NilRoute(t *testing.T) {
	t.Cleanup(func() { SetSkipAuth(false) })

	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)

	request, err := runtime.JSONRequest(http.MethodGet, "/api/pets", nil)
	require.NoError(t, err)

	SetSkipAuth(true)
	p, reqOut, err := ctx.Authorize(request, nil)
	require.NoError(t, err)
	assert.Nil(t, p)
	// Under skip, we return the request as-is rather than nil.
	assert.Equal(t, request, reqOut)
}
