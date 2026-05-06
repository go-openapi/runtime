// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package header

import (
	"net/http"
	"testing"

	"github.com/go-openapi/testify/v2/require"
)

func TestHeader(t *testing.T) {
	hdr := http.Header{
		"x-test": []string{"value"},
	}
	clone := Copy(hdr)
	require.Len(t, clone, len(hdr))
}
