// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package middleware_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-openapi/runtime/middleware"
)

// Test fixtures: extracted to dedup goconst hits across the table-driven cases.
const (
	headerAccept   = "Accept"
	imagePNG       = "image/png"
	imageJPG       = "image/jpg"
	jsonMime       = "application/json"
	jsonUTF8       = "application/json; charset=utf-8"
	textPlainASCII = "text/plain;charset=ascii"

	textPlainUTF8 = "text/plain;charset=utf-8"
	jsonV1        = "application/json;version=1"
	htmlPNGq05    = "text/html, image/png; q=0.5"
	xy            = "x/y"

	textHTML = "text/html"
)

var negotiateContentEncodingTests = []struct {
	s      string
	offers []string
	expect string
}{
	{"", []string{"identity", "gzip"}, "identity"},
	{"*;q=0", []string{"identity", "gzip"}, ""},
	{"gzip", []string{"identity", "gzip"}, "gzip"},
}

func TestNegotiateContentEncoding(t *testing.T) {
	for _, tt := range negotiateContentEncodingTests {
		r := &http.Request{Header: http.Header{"Accept-Encoding": {tt.s}}}
		actual := middleware.NegotiateContentEncoding(r, tt.offers)
		if actual != tt.expect {
			t.Errorf("NegotiateContentEncoding(%q, %#v)=%q, want %q", tt.s, tt.offers, actual, tt.expect)
		}
	}
}

// TestNegotiateContentTypeDefault asserts the v0.30+ default behaviour:
// MIME parameters are honoured by both sides of the match.
//
// Cases inherited from the legacy test suite (which predate the
// parameter-honouring change) keep their original outcomes — they all use
// either bare types or matching params, so honouring vs stripping is a
// no-op for them.
func TestNegotiateContentTypeDefault(t *testing.T) {
	cases := []struct {
		name         string
		acceptHeader string
		offers       []string
		defaultOffer string
		expect       string
	}{
		// --- legacy cases (parameters not in conflict) ---
		{"reject-all via q=0", "text/html, */*;q=0", []string{xy}, "", ""},
		{"wildcard catches anything", "text/html, */*", []string{xy}, "", xy},
		{"first offer wins on tie", "text/html, image/png", []string{textHTML, imagePNG}, "", textHTML},
		{"first offer wins on tie (rev)", "text/html, image/png", []string{imagePNG, textHTML}, "", imagePNG},
		{"non-default match", htmlPNGq05, []string{imagePNG}, "", imagePNG},
		{"q wins over position", htmlPNGq05, []string{textHTML}, "", textHTML},
		{"no match returns default", htmlPNGq05, []string{"foo/bar"}, "", ""},
		{"image/png beats image/* on specificity", "image/png, image/*;q=0.5", []string{imageJPG, imagePNG}, "", imagePNG},
		{"image/* matches jpg", "image/png, image/*;q=0.5", []string{imageJPG}, "", imageJPG},
		{"vendor MIME unmatched (no structural match)", jsonMime, []string{"application/vnd.cia.v1+json"}, "", ""},
		{"java client default", "text/html, image/gif, image/jpeg, *; q=.2, */*; q=.2", []string{jsonMime}, "", jsonMime},

		// --- parameter-honouring matches (offer can satisfy a parametered Accept) ---
		{
			"bare client accept matches param-bearing offer (offer's params satisfy)",
			jsonMime, []string{jsonUTF8, imagePNG}, "",
			jsonUTF8,
		},
		{
			"exact param match",
			jsonUTF8, []string{jsonUTF8, imagePNG}, "",
			jsonUTF8,
		},

		// --- parameter-honouring rejects (the A.4 fix) ---
		{
			// Pre-v0.30 this matched (params stripped). Now: charset values
			// disagree, so the offer no longer satisfies the Accept entry.
			"client-asks-ascii vs offer-utf-8 → no match",
			textPlainASCII, []string{textPlainUTF8}, "",
			"",
		},
		{
			"version mismatch on vendor type → no match",
			"application/json;version=2", []string{jsonV1}, "",
			"",
		},

		// --- parameter case-insensitivity preserved ---
		{
			"value compare case-insensitive (UTF-8 vs utf-8)",
			"text/plain;charset=UTF-8", []string{textPlainUTF8}, "",
			textPlainUTF8,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{headerAccept: {c.acceptHeader}}}
			got := middleware.NegotiateContentType(r, c.offers, c.defaultOffer)
			if got != c.expect {
				t.Errorf("NegotiateContentType(%q, %#v, %q) = %q, want %q",
					c.acceptHeader, c.offers, c.defaultOffer, got, c.expect)
			}
		})
	}
}

// TestNegotiateContentTypeIgnoreParameters covers the explicit opt-out:
// parameters are stripped before matching, restoring the pre-v0.30
// behaviour. Notably, the cases that fail in default mode now succeed.
func TestNegotiateContentTypeIgnoreParameters(t *testing.T) {
	cases := []struct {
		name         string
		acceptHeader string
		offers       []string
		defaultOffer string
		expect       string
	}{
		{
			"client-asks-ascii vs offer-utf-8 (legacy: matches bare)",
			textPlainASCII, []string{textPlainUTF8}, "",
			textPlainUTF8,
		},
		{
			"version mismatch (legacy: matches bare)",
			"application/json;version=2", []string{jsonV1}, "",
			jsonV1,
		},
		// Outcomes that are identical in both modes — sanity checks that
		// IgnoreParameters didn't break the easy cases.
		{
			"bare client accept matches param offer",
			jsonMime, []string{jsonUTF8}, "",
			jsonUTF8,
		},
		{
			"no match returns default (params don't help)",
			imagePNG, []string{textPlainUTF8}, "",
			"",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{headerAccept: {c.acceptHeader}}}
			got := middleware.NegotiateContentType(r, c.offers, c.defaultOffer,
				middleware.WithIgnoreParameters(true),
			)
			if got != c.expect {
				t.Errorf("NegotiateContentType(%q, %#v, %q, WithIgnoreParameters(true)) = %q, want %q",
					c.acceptHeader, c.offers, c.defaultOffer, got, c.expect)
			}
		})
	}
}

// TestNegotiateContentTypeNoAcceptHeader: when Accept is absent the first
// offer is returned regardless of mode. Legacy guarantee, preserved.
func TestNegotiateContentTypeNoAcceptHeader(t *testing.T) {
	r := &http.Request{Header: http.Header{}}
	offers := []string{jsonMime, "text/xml"}
	if got := middleware.NegotiateContentType(r, offers, ""); got != jsonMime {
		t.Errorf("default mode: got %q, want %q", got, jsonMime)
	}
	if got := middleware.NegotiateContentType(r, offers, "", middleware.WithIgnoreParameters(true)); got != jsonMime {
		t.Errorf("ignore mode: got %q, want %q", got, jsonMime)
	}
}

// TestNegotiateContentTypeMultiHeader: multiple Accept header values are
// equivalent to a single comma-joined value (RFC 7230 §3.2.2). The legacy
// test suite's TestContentType_Issue172 relied on this — making the same
// guarantee explicit here.
func TestNegotiateContentTypeMultiHeader(t *testing.T) {
	r := &http.Request{Header: http.Header{headerAccept: {"application/xml", jsonMime}}}
	offers := []string{jsonMime}
	if got := middleware.NegotiateContentType(r, offers, ""); got != jsonMime {
		t.Errorf("got %q, want %q (later Accept value should still match)", got, jsonMime)
	}
}

// ExampleWithIgnoreParameters shows the per-call opt-out for legacy
// parameter-stripping behaviour.
func ExampleWithIgnoreParameters() {
	r := &http.Request{Header: http.Header{headerAccept: {textPlainASCII}}}
	offers := []string{textPlainUTF8}

	// Default: parameters are honoured. The charset values disagree, so
	// no offer matches and we fall back to the default.
	strict := middleware.NegotiateContentType(r, offers, "fallback/default")

	// Opt-out: strip parameters before matching. The bare types agree, so
	// the offer is selected.
	loose := middleware.NegotiateContentType(r, offers, "fallback/default",
		middleware.WithIgnoreParameters(true),
	)

	fmt.Printf("strict=%q\nloose=%q\n", strict, loose)
	// Output:
	// strict="fallback/default"
	// loose="text/plain;charset=utf-8"
}
