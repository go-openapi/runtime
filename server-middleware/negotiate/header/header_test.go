// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package header

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Test fixtures: extracted to dedup goconst hits across the table-driven cases.
const (
	mediaHTML    = "text/html"
	mediaHTMLq05 = "text/html;q=0.5"
	mediaJSON    = "application/json"
	mediaPlain   = "text/plain"
	cdAttachment = "attachment"
	keyFilename  = "filename"
	keyCharset   = "charset"
)

func TestCopy(t *testing.T) {
	t.Run("copies all entries", func(t *testing.T) {
		hdr := http.Header{
			"x-test":       []string{"value"},
			"Content-Type": []string{mediaJSON, mediaPlain},
		}
		clone := Copy(hdr)
		require.Len(t, clone, len(hdr))
		assert.Equal(t, hdr, clone)
	})

	t.Run("returns independent map", func(t *testing.T) {
		hdr := http.Header{"X-Origin": []string{"a"}}
		clone := Copy(hdr)
		clone.Set("X-Origin", "b")
		assert.EqualT(t, "a", hdr.Get("X-Origin"),
			"mutating the clone should not affect the original")
	})

	t.Run("empty header yields empty map", func(t *testing.T) {
		clone := Copy(nil)
		assert.Empty(t, clone)
	})
}

func TestParseTime(t *testing.T) {
	want := time.Date(1994, time.November, 6, 8, 49, 37, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		header string
		want   time.Time
	}{
		{
			name:   "RFC1123 (preferred HTTP-date)",
			header: "Sun, 06 Nov 1994 08:49:37 GMT",
			want:   want,
		},
		{
			name:   "RFC850 (legacy)",
			header: "Sunday, 06-Nov-94 08:49:37 GMT",
			want:   want,
		},
		{
			name:   "ANSIC (legacy)",
			header: "Sun Nov  6 08:49:37 1994",
			want:   want,
		},
		{
			name:   "missing header returns zero value",
			header: "",
			want:   time.Time{},
		},
		{
			name:   "unparseable header returns zero value",
			header: "not a date",
			want:   time.Time{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Date", tc.header)
			}
			got := ParseTime(h, "Date")
			assert.TrueT(t, got.Equal(tc.want), "got %v want %v", got, tc.want)
		})
	}
}

func TestParseList(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header []string
		want   []string
	}{
		{
			name:   "no X-Test header",
			header: nil,
			want:   nil,
		},
		{
			name:   "single value",
			header: []string{"a"},
			want:   []string{"a"},
		},
		{
			name:   "comma separated",
			header: []string{"a,b,c"},
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "trims surrounding whitespace",
			header: []string{" a , b , c "},
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "preserves internal whitespace",
			header: []string{"a b, c d"},
			want:   []string{"a b", "c d"},
		},
		{
			name:   "ignores empty entries",
			header: []string{",a,,b,"},
			want:   []string{"a", "b"},
		},
		{
			name:   "preserves quoted commas",
			header: []string{`"a,b",c`},
			want:   []string{`"a,b"`, "c"},
		},
		{
			name:   "preserves escaped quotes inside quoted strings",
			header: []string{`"a\"b\,c",d`},
			want:   []string{`"a\"b\,c"`, "d"},
		},
		{
			name:   "concatenates multiple header values",
			header: []string{"a,b", "c", " d ,e"},
			want:   []string{"a", "b", "c", "d", "e"},
		},
		{
			name:   "tab counts as whitespace",
			header: []string{"a\t,\tb"},
			want:   []string{"a", "b"},
		},
		{
			name:   "leading whitespace then content",
			header: []string{"  ,  a"},
			want:   []string{"a"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{"X-Test": tc.header}
			got := ParseList(h, "X-Test")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseValueAndParams(t *testing.T) {
	for _, tc := range []struct {
		name       string
		header     string
		wantValue  string
		wantParams map[string]string
	}{
		{
			name:       "value only, lowercased",
			header:     "Application/JSON",
			wantValue:  mediaJSON,
			wantParams: map[string]string{},
		},
		{
			name:       "value with single param",
			header:     "text/plain; charset=utf-8",
			wantValue:  mediaPlain,
			wantParams: map[string]string{keyCharset: "utf-8"},
		},
		{
			name:       "param keys lowercased, values preserved",
			header:     "Text/Plain; CharSet=UTF-8",
			wantValue:  mediaPlain,
			wantParams: map[string]string{keyCharset: "UTF-8"},
		},
		{
			name:       "multiple params",
			header:     "form-data; name=\"file\"; filename=\"a.txt\"",
			wantValue:  "form-data",
			wantParams: map[string]string{"name": "file", keyFilename: "a.txt"},
		},
		{
			name:       "quoted param with escape",
			header:     `attachment; filename="a\"b.txt"`,
			wantValue:  cdAttachment,
			wantParams: map[string]string{keyFilename: `a"b.txt`},
		},
		{
			// `\\` in the quoted value: first '\' starts an escape (consuming
			// the next '\'), exercising the inner `case b == '\\'` branch in
			// the escape loop of expectTokenOrQuoted.
			name:       "quoted param with double-backslash escape",
			header:     `attachment; filename="a\\b"`,
			wantValue:  cdAttachment,
			wantParams: map[string]string{keyFilename: `a\b`},
		},
		{
			// Two non-adjacent escape sequences inside one quoted value: the
			// second '\' is reached while `escape == false`, triggering the
			// inner `case b == '\\'` re-entry branch.
			name:       "quoted param with two escape sequences",
			header:     `attachment; filename="\AB\CD"`,
			wantValue:  cdAttachment,
			wantParams: map[string]string{keyFilename: `ABCD`},
		},
		{
			// Unterminated quoted string with an in-progress escape: the
			// inner loop in expectTokenOrQuoted exits without finding the
			// closing quote and returns ("", "").
			name:       "unterminated quoted param with escape stops parsing",
			header:     `attachment; filename="a\b`,
			wantValue:  cdAttachment,
			wantParams: map[string]string{},
		},
		{
			// Unterminated quoted string without any escape: the outer loop
			// in expectTokenOrQuoted exits without finding the closing quote.
			name:       "unterminated quoted param stops parsing",
			header:     `attachment; filename="abc`,
			wantValue:  cdAttachment,
			wantParams: map[string]string{},
		},
		{
			name:       "tolerant of extra whitespace around semicolons",
			header:     "text/html ;  charset=utf-8 ",
			wantValue:  mediaHTML,
			wantParams: map[string]string{keyCharset: "utf-8"},
		},
		{
			name:       "whitespace around '=' stops param parsing (parser is strict here)",
			header:     "text/html; charset = utf-8",
			wantValue:  mediaHTML,
			wantParams: map[string]string{},
		},
		{
			name:       "empty header returns empty value",
			header:     "",
			wantValue:  "",
			wantParams: map[string]string{},
		},
		{
			name:       "missing param value stops parsing without error",
			header:     "text/plain; charset",
			wantValue:  mediaPlain,
			wantParams: map[string]string{},
		},
		{
			name:       "missing param key stops parsing without error",
			header:     "text/plain; =utf-8",
			wantValue:  mediaPlain,
			wantParams: map[string]string{},
		},
		{
			name:       "empty quoted param value stops parsing without error",
			header:     `text/plain; charset=""; boundary=foo`,
			wantValue:  mediaPlain,
			wantParams: map[string]string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Content-Type", tc.header)
			}
			value, params := ParseValueAndParams(h, "Content-Type")
			assert.EqualT(t, tc.wantValue, value)
			assert.Equal(t, tc.wantParams, params)
		})
	}
}

// TestParseAccept exercises the streaming parser used for Accept and
// Accept-Encoding headers (the canonical-key variant).
func TestParseAccept(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header []string
		want   []AcceptSpec
	}{
		{
			name:   "no Accept header",
			header: nil,
			want:   nil,
		},
		{
			name:   "single value defaults q to 1.0",
			header: []string{mediaHTML},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 1.0}},
		},
		{
			name:   "explicit q parameter",
			header: []string{mediaHTMLq05},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0.5}},
		},
		{
			name:   "q=1 parsed as 1.0",
			header: []string{"text/html;q=1"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 1.0}},
		},
		{
			name:   "q=0 parsed as 0",
			header: []string{"text/html;q=0"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0}},
		},
		{
			name:   "q=1.0 parsed as 1.0",
			header: []string{"text/html;q=1.0"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 1.0}},
		},
		{
			name:   "q=.5 parsed as 0.5",
			header: []string{"text/html;q=.5"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0.5}},
		},
		{
			name:   "multiple types, q ordering preserved",
			header: []string{"text/html;q=0.8, application/json;q=0.9, */*;q=0.1"},
			want: []AcceptSpec{
				{Value: mediaHTML, Q: 0.8},
				{Value: mediaJSON, Q: 0.9},
				{Value: "*/*", Q: 0.1},
			},
		},
		{
			name:   "ignores unknown parameters before q",
			header: []string{"text/html;level=1;q=0.4"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0.4}},
		},
		{
			name:   "ignores unknown parameters after q (consumed by leading-tokens loop)",
			header: []string{"text/html;q=0.4;extra=ignored"},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0.4}},
		},
		{
			name:   "Accept-Encoding style tokens",
			header: []string{"gzip, deflate, br;q=0.9"},
			want: []AcceptSpec{
				{Value: "gzip", Q: 1.0},
				{Value: "deflate", Q: 1.0},
				{Value: "br", Q: 0.9},
			},
		},
		{
			name:   "concatenates across multiple header values",
			header: []string{mediaHTMLq05, mediaJSON},
			want: []AcceptSpec{
				{Value: mediaHTML, Q: 0.5},
				{Value: mediaJSON, Q: 1.0},
			},
		},
		{
			name:   "malformed leading token skipped",
			header: []string{",, text/html"},
			want:   nil,
		},
		{
			name:   "invalid q-value drops entry",
			header: []string{"text/html;q=bogus"},
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// ParseAccept reads via direct map indexing, not Get; use the
			// canonical key explicitly.
			h := http.Header{"Accept": tc.header}
			got := ParseAccept(h, "Accept")
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestParseAccept2 covers the alternate parser that goes through ParseList
// and parseValueAndParams. It is more permissive about formatting than
// ParseAccept.
func TestParseAccept2(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header []string
		want   []AcceptSpec
	}{
		{
			name:   "no Accept header (alt parser)",
			header: nil,
			want:   nil,
		},
		{
			name:   "single value defaults q to 1.0",
			header: []string{mediaHTML},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 1.0}},
		},
		{
			name:   "explicit q parameter",
			header: []string{mediaHTMLq05},
			want:   []AcceptSpec{{Value: mediaHTML, Q: 0.5}},
		},
		{
			name:   "multiple types preserved in order",
			header: []string{"text/html;q=0.8, application/json"},
			want: []AcceptSpec{
				{Value: mediaHTML, Q: 0.8},
				{Value: mediaJSON, Q: 1.0},
			},
		},
		{
			name:   "rejects entries with negative q",
			header: []string{"text/html;q=bogus, application/json"},
			// expectQuality returns -1 for "bogus", which drops the entry.
			want: []AcceptSpec{{Value: mediaJSON, Q: 1.0}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// ParseAccept2 routes through ParseList, which canonicalises the
			// key — using the raw "Accept" form is fine here.
			h := http.Header{"Accept": tc.header}
			got := ParseAccept2(h, "Accept")
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestExpectQuality pins the behaviour of the q-value mini-parser
// (RFC 7231 §5.3.1). It is exercised indirectly by ParseAccept; the table
// below targets each branch directly.
func TestExpectQuality(t *testing.T) {
	for _, tc := range []struct {
		input   string
		wantQ   float64
		wantRem string
	}{
		{"", -1, ""},
		{"0", 0, ""},
		{"1", 1, ""},
		{".5", 0.5, ""},
		{"0.5", 0.5, ""},
		{"1.0", 1, ""},
		{"0.123", 0.123, ""},
		{"0.5,next", 0.5, ",next"},
		{"x", -1, ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			q, rem := expectQuality(tc.input)
			assert.InDeltaT(t, tc.wantQ, q, 1e-9)
			assert.EqualT(t, tc.wantRem, rem)
		})
	}
}
