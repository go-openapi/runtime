// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package mediatype

import (
	"strings"
	"testing"
)

// Test-only constants pulled out for goconst. The `jsonMime` and
// `starStar` constants are shared with the rest of the in-package
// test corpus (mediatype_test.go).
const (
	testMTAppPrefix = "application/"
	testMTSubJSON   = "/json"
	testMTAppStar   = "application/*"
)

// FuzzParse exercises [Parse] with arbitrary input. The invariant
// is: Parse must not panic, hang, or return a non-zero MediaType
// alongside a non-nil error.
//
// Lens 4 (header parsing) of the security scrub:
// .claude/plans/security-scrub.md.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"",
		" ",
		jsonMime,
		jsonMime + "; charset=utf-8",
		jsonMime + ";q=0.5",
		jsonMime + " ; charset=utf-8 ; q=0.5",
		"application/problem+json",
		"application/vnd.api+json; version=1",
		"text/plain; charset=\"utf-8\"",
		"text/plain; charset=\"utf\\\"8\"",
		starStar,
		testMTAppStar,
		"application/json,text/xml", // multi-entry — Parse is single-only
		jsonMime + "; q=2.0",        // invalid q
		jsonMime + "; q=-1",         // invalid q
		jsonMime + "; q=abc",        // invalid q
		testMTAppPrefix,
		testMTSubJSON,
		"application",
		jsonMime + "/extra",
		";charset=utf-8",
		jsonMime + "; ;",
		jsonMime + ";;",
		jsonMime + "; charset=",
		jsonMime + "; charset",
		jsonMime + "; charset=\xff\xfe",
		jsonMime + "+",
		"application/+json",
		"application/json+\x00",
		strings.Repeat("a", 4096), // long type
		jsonMime + "; " + strings.Repeat("x=y;", 256),       // many params
		jsonMime + "; charset=" + strings.Repeat("a", 4096), // long value
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		mt, err := Parse(in)
		if err != nil {
			// Error path: zero MediaType expected.
			if mt.Type != "" || mt.Subtype != "" || mt.Suffix != "" || len(mt.Params) != 0 {
				t.Fatalf("Parse(%q) returned (mt=%+v, err=%v) — non-zero MediaType with error", in, mt, err)
			}
			return
		}
		// Success path: type and subtype must be non-empty.
		if mt.Type == "" || mt.Subtype == "" {
			t.Fatalf("Parse(%q) succeeded with empty Type/Subtype: %+v", in, mt)
		}
		// Q must be in [0, 1] when no q-value supplied (default 1.0)
		// or when one was; we don't differentiate here, just that
		// it's a valid float in a sane range.
		if mt.Q < 0 || mt.Q > 1 {
			t.Fatalf("Parse(%q) Q=%v out of [0,1]", in, mt.Q)
		}
	})
}

// FuzzMatchFirst exercises [MatchFirst] with arbitrary actual
// values against a fixed allowed list. The invariant is: must
// not panic, hang, or return ok=true with a zero MediaType.
//
// We fuzz the actual rather than both sides because the allowed
// list is typically a server-configured offer set (operator-trusted)
// while the actual is the client-supplied Content-Type / Accept
// header (untrusted).
func FuzzMatchFirst(f *testing.F) {
	allowed := []string{
		jsonMime,
		"application/xml",
		"text/plain",
		"application/vnd.api+json",
		starStar,
	}

	seeds := []string{
		"",
		jsonMime,
		jsonMime + "; charset=utf-8",
		"application/problem+json",
		"text/plain",
		"application/octet-stream",
		"",
		"\x00",
		"\xff\xfe",
		strings.Repeat("a", 4096),
		testMTAppPrefix + strings.Repeat("x", 1024),
		testMTSubJSON,
		testMTAppPrefix,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, actual string) {
		mt, ok, err := MatchFirst(allowed, actual, AllowSuffix())
		if ok && (mt.Type == "" || mt.Subtype == "") {
			t.Fatalf("MatchFirst(%q) returned ok=true with empty MediaType: %+v", actual, mt)
		}
		if !ok && mt.Type != "" {
			t.Fatalf("MatchFirst(%q) returned ok=false with non-zero MediaType: %+v", actual, mt)
		}
		// err may be set for malformed actuals; not a fault.
		_ = err
	})
}

// FuzzParseAccept exercises [ParseAccept] with arbitrary Accept
// headers. The invariant is: must not panic, hang, or return a
// non-empty Set with entries that fail their own invariants
// (Type/Subtype non-empty; Q in [0,1]).
func FuzzParseAccept(f *testing.F) {
	seeds := []string{
		"",
		jsonMime,
		jsonMime + "; q=0.5",
		"application/json, text/xml; q=0.8, */*; q=0.1",
		"text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
		jsonMime + "; q=2.0", // invalid q
		jsonMime + "; q=-1",  // invalid q
		"application/json,, text/plain",
		jsonMime + ";q=0.5;charset=utf-8",
		"," + strings.Repeat("a", 1024),
		strings.Repeat(",", 256),
		strings.Repeat("application/json,", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		set := ParseAccept(in)
		for i, mt := range set {
			if mt.Type == "" || mt.Subtype == "" {
				t.Fatalf("ParseAccept(%q)[%d] empty Type/Subtype: %+v", in, i, mt)
			}
			if mt.Q < 0 || mt.Q > 1 {
				t.Fatalf("ParseAccept(%q)[%d] Q=%v out of [0,1]", in, i, mt.Q)
			}
		}
	})
}
