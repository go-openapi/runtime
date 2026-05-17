// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzBindForm exercises the full BindForm parse path with arbitrary
// multipart bodies. Invariants: must not panic, hang, or return
// fatal=true alongside err=nil.
//
// Security scrub Lens 3 / L3.8 — fuzz coverage for the form-binding
// surface.
func FuzzBindForm(f *testing.F) {
	const boundary = "FUZZBOUND"
	ct := "multipart/form-data; boundary=" + boundary

	seeds := [][]byte{
		// Well-formed single text part.
		[]byte("--" + boundary + "\r\n" +
			`Content-Disposition: form-data; name="x"` + "\r\n\r\n" +
			"v\r\n" +
			"--" + boundary + "--\r\n"),
		// Well-formed single file part.
		[]byte("--" + boundary + "\r\n" +
			`Content-Disposition: form-data; name="f"; filename="t.txt"` + "\r\n" +
			"Content-Type: text/plain\r\n\r\n" +
			"data\r\n" +
			"--" + boundary + "--\r\n"),
		// Empty body.
		nil,
		// Only the closing boundary.
		[]byte("--" + boundary + "--\r\n"),
		// Truncated body (no closing boundary).
		[]byte("--" + boundary + "\r\n" +
			`Content-Disposition: form-data; name="x"` + "\r\n\r\n"),
		// Adversarial filename (long).
		[]byte("--" + boundary + "\r\n" +
			`Content-Disposition: form-data; name="f"; filename="` +
			strings.Repeat("a", 4096) + `"` + "\r\n\r\n" +
			"x\r\n" +
			"--" + boundary + "--\r\n"),
		// Garbage that doesn't start with a boundary.
		[]byte("not-a-multipart-body"),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))
		r.Header.Set("Content-Type", ct)

		fatal, err := BindForm(r,
			BindFormFile("f", false, func(_ multipart.File, _ *multipart.FileHeader) error { return nil }),
		)

		if fatal && err == nil {
			t.Fatalf("BindForm returned fatal=true with err=nil for body %q", body)
		}
	})
}

// FuzzBindFormFilename targets the filename-cap path specifically.
// It feeds an arbitrary filename through a synthetic well-formed
// multipart body and asserts the bound *FileHeader.Filename length
// never exceeds DefaultMaxUploadFilenameLength.
//
// Security scrub Lens 3 / L3.1 + L3.8.
func FuzzBindFormFilename(f *testing.F) {
	seeds := []string{
		"normal.txt",
		"",
		strings.Repeat("a", DefaultMaxUploadFilenameLength),     // exactly at cap
		strings.Repeat("a", DefaultMaxUploadFilenameLength+1),   // one over
		strings.Repeat("a", DefaultMaxUploadFilenameLength*100), // way over
		"a/b/c.txt",
		"../etc/passwd",
		"\x00",
		"file\r\nContent-Type: forged",
		string([]byte{0xff, 0xfe}),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	const boundary = "FUZZBOUND"

	f.Fuzz(func(t *testing.T, filename string) {
		// Build a well-formed multipart wrapper around the fuzzed
		// filename. %q quote-escapes so the wire stays parseable;
		// the bytes BindForm sees as Filename are the same fuzz
		// input after the stdlib parser decodes the quoted-string.
		body := fmt.Sprintf(
			"--%s\r\n"+
				`Content-Disposition: form-data; name="f"; filename=%q`+"\r\n"+
				"Content-Type: application/octet-stream\r\n"+
				"\r\n"+
				"data\r\n"+
				"--%s--\r\n",
			boundary, filename, boundary,
		)
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/",
			strings.NewReader(body))
		r.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

		var bound *multipart.FileHeader
		fatal, err := BindForm(r,
			BindFormFile("f", false, func(_ multipart.File, h *multipart.FileHeader) error {
				bound = h
				return nil
			}),
		)

		if fatal && err == nil {
			t.Fatalf("fatal=true with err=nil for filename %q", filename)
		}
		if err == nil && bound != nil {
			if len(bound.Filename) > DefaultMaxUploadFilenameLength {
				t.Fatalf("BindForm bound a file with filename length %d > cap %d (filename=%q)",
					len(bound.Filename), DefaultMaxUploadFilenameLength, bound.Filename)
			}
		}
	})
}
