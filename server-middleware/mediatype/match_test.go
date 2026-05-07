// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype_test

import (
	"testing"

	"github.com/go-openapi/runtime/server-middleware/mediatype"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const (
	jsonMime          = "application/json"
	yamlMime          = "application/x-yaml"
	htmlMime          = "text/html"
	octetMime         = "application/octet-stream"
	jpegMime          = "image/jpeg"
	imagePNG          = "image/png"
	textPlain         = "text/plain"
	textPlainUTF8     = "text/plain;charset=utf-8"
	textPlainParamSrv = "text/plain; charset=utf-8"
)

// TestMatch covers the matching primitive used by the server-side
// Content-Type validator. Rows ported from middleware/validation_test.go
// (TestValidateContentType) plus new rows for the (MediaType, bool, error)
// distinctions only the primitive surfaces.
func TestMatch(t *testing.T) {
	t.Run("matches and rejections", func(t *testing.T) {
		cases := []struct {
			name    string
			actual  string
			allowed []string
			wantOK  bool
		}{
			{"exact bare match", jsonMime, []string{jsonMime}, true},
			{"no match in list", jsonMime, []string{yamlMime, htmlMime}, false},
			{"client param, allowed bare (with space)", "text/html; charset=utf-8", []string{htmlMime}, true},
			{"client param, allowed bare (no space)", "text/html;charset=utf-8", []string{htmlMime}, true},
			{"unrelated types", "text/html;           charset=utf-8", []string{jsonMime}, false},
			{"subtype wildcard on allowed", octetMime, []string{jpegMime, "application/*"}, true},
			{"full wildcard on allowed", imagePNG, []string{"*/*", jsonMime}, true},

			// Regression for https://github.com/go-openapi/runtime/issues/136 —
			// allowed entries with MIME parameters must not block matching clients.
			{"#136 client bare, allowed has params", textPlain, []string{textPlainParamSrv}, true},
			{"#136 client param differs", "text/plain;blah=true", []string{textPlainParamSrv}, false},
			{"#136 client params, allowed bare", textPlainUTF8, []string{textPlain}, true},
			{"#136 exact param match", textPlainUTF8, []string{textPlainUTF8}, true},
			{"#136 client param value case-insensitive", "text/plain;charset=UTF-8", []string{textPlainUTF8}, true},
			{"#136 conflicting param values", textPlainUTF8, []string{"text/plain;charset=ascii"}, false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				_, ok, err := mediatype.MatchFirst(c.allowed, c.actual)
				require.NoError(t, err)
				assert.EqualT(t, c.wantOK, ok)
			})
		}
	})

	t.Run("malformed actual surfaces ErrMalformed", func(t *testing.T) {
		// These rows used to be in TestValidateContentType under the
		// "*errors.Validation" expectation. Match exposes the cause so
		// callers can distinguish 400 from 415 if they want to.
		malformed := []struct {
			name   string
			actual string
		}{
			{"empty", ""},
			{"unparseable", "application("},
			{"bad parameter", "application/json;char*"},
		}
		for _, c := range malformed {
			t.Run(c.name, func(t *testing.T) {
				_, ok, err := mediatype.MatchFirst([]string{jsonMime}, c.actual)
				assert.False(t, ok)
				require.Error(t, err)
				assert.ErrorIs(t, err, mediatype.ErrMalformed)
			})
		}
	})

	t.Run("empty allowed: (_, false, nil)", func(t *testing.T) {
		// Match is a primitive — empty constraints means no match. The
		// "empty list = accept anything" policy lives in the caller.
		_, ok, err := mediatype.MatchFirst(nil, jsonMime)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("malformed allowed entries are skipped", func(t *testing.T) {
		// An entry in `allowed` that fails to parse cannot match any
		// well-formed actual, so it is silently skipped. Other entries
		// in the list still get a chance to match.
		t.Run("skipped, fall through to next match", func(t *testing.T) {
			_, ok, err := mediatype.MatchFirst([]string{"garbage(", jsonMime}, jsonMime)
			require.NoError(t, err)
			assert.True(t, ok)
		})
		t.Run("only malformed entries: no match, no error", func(t *testing.T) {
			_, ok, err := mediatype.MatchFirst([]string{"garbage(", "also-bad"}, jsonMime)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})

	t.Run("matched entry returned as parsed MediaType", func(t *testing.T) {
		got, ok, err := mediatype.MatchFirst([]string{textPlainParamSrv}, textPlain)
		require.NoError(t, err)
		require.True(t, ok)
		assert.EqualT(t, "text", got.Type)
		assert.EqualT(t, "plain", got.Subtype)
		assert.EqualT(t, "utf-8", got.Params["charset"])
	})
}
