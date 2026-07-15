// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/internal/testing/petstore"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func TestRouteInfoSetsRequestPattern(t *testing.T) {
	spec, api := petstore.NewAPI(t)
	ctx := NewContext(spec, api, nil)
	ctx.router = DefaultRouter(spec, ctx.api)

	t.Run("sets Request.Pattern to the matched route", func(t *testing.T) {
		request, err := runtime.JSONRequest(http.MethodGet, "/api/pets/123", nil)
		require.NoError(t, err)

		mr, reqWithCtx, ok := ctx.RouteInfo(request)
		require.True(t, ok)
		require.NotNil(t, reqWithCtx)

		assert.Equal(t, mr.BasePath+mr.PathPattern, reqWithCtx.Pattern)
		assert.True(t, strings.HasSuffix(reqWithCtx.Pattern, "/pets/{id}"), reqWithCtx.Pattern)
		assert.Equal(t, reqWithCtx.Pattern, request.Pattern)
	})

	t.Run("leaves Request.Pattern empty when no route matches", func(t *testing.T) {
		request, err := runtime.JSONRequest(http.MethodGet, "/api/not-a-route", nil)
		require.NoError(t, err)

		_, _, ok := ctx.RouteInfo(request)
		require.False(t, ok)
		assert.Empty(t, request.Pattern)
	})
}
