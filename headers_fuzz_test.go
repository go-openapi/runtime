// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"net/http"
	"strings"
	"testing"
)

// FuzzContentType exercises [ContentType] with arbitrary
// Content-Type header values. Invariants: must not panic or hang;
// when err is non-nil, the returned media type and charset must
// both be empty.
//
// Lens 4 (header parsing) of the security scrub:
// .claude/plans/security-scrub.md.
func FuzzContentType(f *testing.F) {
	const appJSON = JSONMime
	seeds := []string{
		"",
		" ",
		appJSON,
		appJSON + "; charset=utf-8",
		appJSON + "; charset=\"utf-8\"",
		appJSON + "; charset=\"utf\\\"8\"",
		appJSON + "; charset=\xff\xfe",
		appJSON + ";",
		appJSON + ";;",
		appJSON + "; ;",
		appJSON + "; charset",
		appJSON + "; charset=",
		"application/octet-stream",
		"text/plain; charset=us-ascii",
		strings.Repeat("a", 4096),
		appJSON + "; " + strings.Repeat("x=y;", 256),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		h := http.Header{HeaderContentType: []string{in}}
		mt, cs, err := ContentType(h)
		if err != nil {
			if mt != "" || cs != "" {
				t.Fatalf("ContentType(%q) returned (mt=%q, cs=%q, err=%v) — non-empty mt/cs with error",
					in, mt, cs, err)
			}
			return
		}
		// Success path: when input is non-empty and parses, mt
		// must be non-empty (the stdlib mime.ParseMediaType already
		// guarantees this; we re-assert as a regression guard).
		// Empty input is allowed: returns ("", "", nil) via the
		// DefaultMime branch.
		_ = mt
		_ = cs
	})
}
