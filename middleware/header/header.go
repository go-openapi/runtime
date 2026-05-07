// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package header forwards to the relocated implementation at
// [github.com/go-openapi/runtime/server-middleware/negotiate/header].
//
// Deprecated: this package was unintentionally exposed and has moved to
// [github.com/go-openapi/runtime/server-middleware/negotiate/header].
//
// The shim preserves the public surface so existing imports keep
// compiling against v0.30.x; new code should target the new path.
package header

import (
	"net/http"
	"time"

	upstream "github.com/go-openapi/runtime/server-middleware/negotiate/header"
)

// AcceptSpec describes an entry parsed from an Accept-style header.
//
// Deprecated: see package documentation.
type AcceptSpec = upstream.AcceptSpec

// Copy returns a shallow copy of the header.
//
// Deprecated: see package documentation.
func Copy(header http.Header) http.Header {
	return upstream.Copy(header)
}

// ParseList parses a comma separated list of values.
//
// Commas are ignored in quoted strings. Quoted values are not unescaped or
// unquoted. Whitespace is trimmed.
//
// Deprecated: see package documentation.
func ParseList(header http.Header, key string) []string {
	return upstream.ParseList(header, key)
}

// ParseTime parses the header as time.
//
// The zero value is returned if the header is not present or there is an
// error parsing the header.
//
// Deprecated: see package documentation.
func ParseTime(header http.Header, key string) time.Time {
	return upstream.ParseTime(header, key)
}

// ParseValueAndParams parses a comma separated list of values with optional
// semicolon separated name-value pairs.
//
// Content-Type and Content-Disposition headers are in this format.
//
// Deprecated: see package documentation.
func ParseValueAndParams(header http.Header, key string) (string, map[string]string) {
	return upstream.ParseValueAndParams(header, key)
}

// ParseAccept parses Accept* headers.
//
// Deprecated: see package documentation.
func ParseAccept(header http.Header, key string) []AcceptSpec {
	return upstream.ParseAccept(header, key)
}

// ParseAccept2 parses Accept* headers (alternate parser).
//
// Deprecated: see package documentation.
func ParseAccept2(header http.Header, key string) (specs []AcceptSpec) {
	return upstream.ParseAccept2(header, key)
}
