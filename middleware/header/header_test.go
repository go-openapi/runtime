// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package header_test

import (
	"net/http"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	// shim under test
	header "github.com/go-openapi/runtime/middleware/header"
	upstream "github.com/go-openapi/runtime/server-middleware/negotiate/header"
)

// TestShimWiring is a smoke test: it asserts that every exported symbol
// re-exported from middleware/header forwards to the relocated package.
// Edge-case behaviour is exhaustively covered in the upstream package
// (server-middleware/negotiate/header) — the assertions here only need
// to be specific enough to prove the call landed there.
func TestShimWiring(t *testing.T) {
	t.Run("AcceptSpec is a type alias for upstream.AcceptSpec", func(t *testing.T) {
		// Type alias means a value of one is assignable to the other
		// without conversion. If the shim re-declared the struct we'd
		// need an explicit cast and this would not compile. The explicit
		// type annotations are the assertion — do not let inference
		// erase them.
		//nolint:staticcheck // ST1023: explicit annotations prove the alias
		var s header.AcceptSpec = upstream.AcceptSpec{Value: "x", Q: 1.0}
		//nolint:staticcheck // ST1023: explicit annotations prove the alias
		var u upstream.AcceptSpec = s
		assert.EqualT(t, "x", u.Value)
		assert.InDeltaT(t, 1.0, u.Q, 0)
	})

	t.Run("Copy forwards", func(t *testing.T) {
		in := http.Header{"X-Test": []string{"v"}}
		got := header.Copy(in)
		require.Len(t, got, 1)
		assert.EqualT(t, "v", got.Get("X-Test"))
	})

	t.Run("ParseList forwards", func(t *testing.T) {
		got := header.ParseList(http.Header{"X-Test": []string{"a, b"}}, "X-Test")
		assert.Equal(t, []string{"a", "b"}, got)
	})

	t.Run("ParseTime forwards", func(t *testing.T) {
		h := http.Header{}
		h.Set("Date", "Sun, 06 Nov 1994 08:49:37 GMT")
		got := header.ParseTime(h, "Date")
		assert.EqualT(t, 1994, got.Year())
	})

	t.Run("ParseValueAndParams forwards", func(t *testing.T) {
		h := http.Header{}
		h.Set("Content-Type", "text/plain; charset=utf-8")
		value, params := header.ParseValueAndParams(h, "Content-Type")
		assert.EqualT(t, "text/plain", value)
		assert.EqualT(t, "utf-8", params["charset"])
	})

	t.Run("ParseAccept forwards", func(t *testing.T) {
		got := header.ParseAccept(http.Header{"Accept": []string{"text/html;q=0.5"}}, "Accept")
		require.Len(t, got, 1)
		assert.EqualT(t, "text/html", got[0].Value)
		assert.InDeltaT(t, 0.5, got[0].Q, 1e-9)
	})

	t.Run("ParseAccept2 forwards", func(t *testing.T) {
		got := header.ParseAccept2(http.Header{"Accept": []string{"text/html;q=0.5"}}, "Accept")
		require.Len(t, got, 1)
		assert.EqualT(t, "text/html", got[0].Value)
		assert.InDeltaT(t, 0.5, got[0].Q, 1e-9)
	})
}
