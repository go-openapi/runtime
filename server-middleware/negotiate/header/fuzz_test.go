// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package header

import (
	"net/http"
	"strings"
	"testing"
)

const testHdrAccept = "Accept"

// FuzzParseValueAndParams exercises [parseValueAndParams] (the
// string-level version of [ParseValueAndParams]) with arbitrary
// input. Invariants: must not panic, hang, or return a non-empty
// params map with empty keys.
//
// Lens 4 (header parsing) of the security scrub:
// .claude/plans/security-scrub.md.
func FuzzParseValueAndParams(f *testing.F) {
	seeds := []string{
		"",
		" ",
		"application/json",
		"application/json; charset=utf-8",
		"application/json; charset=\"utf-8\"",
		"application/json; charset=\"utf\\\"8\"",
		"application/json;",
		"application/json;;",
		"application/json; ; charset=utf-8",
		"application/json; charset",
		"application/json; charset=",
		"application/json; =utf-8",
		"application/json; charset=utf-8; q=0.5",
		"text/plain;param1=v1;param2=\"v 2\"",
		"text/plain; param=\"\\\"\"",
		"text/plain; param=\"\xff\xfe\"",
		strings.Repeat("a", 1024) + "/json",
		"application/json; " + strings.Repeat("k=v;", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		value, params := parseValueAndParams(in)
		// Invariants: empty keys forbidden in params; if value is
		// empty, params must also be empty (the function bails
		// out before populating params).
		if value == "" && len(params) != 0 {
			t.Fatalf("parseValueAndParams(%q) → value=\"\" but params=%v", in, params)
		}
		for k := range params {
			if k == "" {
				t.Fatalf("parseValueAndParams(%q) emitted empty param key; params=%v", in, params)
			}
		}
	})
}

// FuzzParseAccept exercises [ParseAccept] via a real http.Header
// populated with the fuzzed input. Invariants: must not panic,
// hang, or return AcceptSpec entries with empty Value or Q
// outside [0, 1].
func FuzzParseAccept(f *testing.F) {
	seeds := []string{
		"",
		"application/json",
		"text/html, application/xhtml+xml, application/xml;q=0.9, */*;q=0.8",
		"application/json; q=0.5, text/xml; q=0.7",
		"application/json;q=2.0",
		"application/json;q=-1",
		"application/json,, text/plain",
		"application/json,application/xml,text/plain",
		"application/json;charset=utf-8;q=0.5",
		"application/json;q=foo",
		strings.Repeat("application/json,", 256),
		strings.Repeat(",", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		h := http.Header{testHdrAccept: []string{in}}
		specs := ParseAccept(h, testHdrAccept)
		for i, sp := range specs {
			if sp.Value == "" {
				t.Fatalf("ParseAccept(%q)[%d] empty Value", in, i)
			}
			if sp.Q < 0 || sp.Q > 1 {
				t.Fatalf("ParseAccept(%q)[%d] Q=%v out of [0,1]", in, i, sp.Q)
			}
		}
	})
}

// FuzzParseList exercises [ParseList] (comma-separated header
// list parser). Invariants: no panic, no empty entries.
func FuzzParseList(f *testing.F) {
	seeds := []string{
		"",
		"a",
		"a,b,c",
		"a, b, c",
		" a , b , c ",
		"a,,b",
		",a",
		"a,",
		"a,\"b,c\",d",
		"a,\"b\\\"c\",d",
		strings.Repeat("a,", 256),
		strings.Repeat(",", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		h := http.Header{"X-List": []string{in}}
		out := ParseList(h, "X-List")
		for i, v := range out {
			if v == "" {
				t.Fatalf("ParseList(%q)[%d] empty entry", in, i)
			}
		}
	})
}
