// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package request

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/v2/assert"
)

// TestEscapeQuotes_StripsCRLF verifies that escapeQuotes neutralises
// CR / LF in Content-Disposition parameter values, preventing
// header-injection through attacker-influenced field names or
// filenames. Mirrors the known stdlib gap golang/go#19038.
//
// Security scrub Lens 3 / L3.2.
func TestEscapeQuotes_StripsCRLF(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "embedded CR",
			in:   "file\rname.txt",
			want: "file_name.txt",
		},
		{
			name: "embedded LF",
			in:   "file\nname.txt",
			want: "file_name.txt",
		},
		{
			name: "embedded CRLF",
			in:   "file\r\nname.txt",
			want: "file__name.txt",
		},
		{
			name: "CRLF + injected header",
			in:   "evil.txt\r\nContent-Type: forged",
			want: "evil.txt__Content-Type: forged",
		},
		{
			name: "no control chars",
			in:   "regular.txt",
			want: "regular.txt",
		},
		{
			name: "still escapes quote",
			in:   `with"quote.txt`,
			want: `with\"quote.txt`,
		},
		{
			name: "still escapes backslash",
			in:   `with\backslash.txt`,
			want: `with\\backslash.txt`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := escapeQuotes(c.in)
			assert.Equal(t, c.want, got)
			// Belt-and-braces: result must contain no literal CR/LF
			// regardless of how the input was assembled.
			assert.False(t, strings.ContainsAny(got, "\r\n"),
				"output retained a CR or LF: %q", got)
		})
	}
}
