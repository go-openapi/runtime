// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"mime"
	"net/http"
	"testing"

	"github.com/go-openapi/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseContentType(t *testing.T) {
	_, _, reason1 := mime.ParseMediaType("application(")
	_, _, reason2 := mime.ParseMediaType("application/json;char*")
	data := []struct {
		hdr, mt, cs string
		err         *errors.ParseError
	}{
		{"application/json", "application/json", "", nil},
		{"text/html; charset=utf-8", "text/html", "utf-8", nil},
		{"text/html;charset=utf-8", "text/html", "utf-8", nil},
		{"", "application/octet-stream", "", nil},
		{"text/html;           charset=utf-8", "text/html", "utf-8", nil},
		{"application(", "", "", errors.NewParseError("Content-Type", "header", "application(", reason1)},
		{"application/json;char*", "", "", errors.NewParseError("Content-Type", "header", "application/json;char*", reason2)},
	}

	headers := http.Header(map[string][]string{})
	for _, v := range data {
		if v.hdr != "" {
			headers.Set("Content-Type", v.hdr)
		} else {
			headers.Del("Content-Type")
		}
		ct, cs, err := ContentType(headers)
		if v.err == nil {
			require.NoError(t, err, "input: %q, err: %v", v.hdr, err)
		} else {
			require.Error(t, err, "input: %q", v.hdr)
			assert.IsTypef(t, &errors.ParseError{}, err, "input: %q", v.hdr) //nolint: testifylint // ErrorAs doesn't work and ErrorIs doesn't fit
			assert.Equal(t, v.err.Error(), err.Error(), "input: %q", v.hdr)
		}
		assert.Equal(t, v.mt, ct, "input: %q", v.hdr)
		assert.Equal(t, v.cs, cs, "input: %q", v.hdr)
	}
}
