// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-openapi/runtime"
)

func TestResponse(t *testing.T) {
	under := new(http.Response)
	under.Status = "the status message"
	under.StatusCode = 392
	under.Header = make(http.Header)
	under.Header.Set("Blah", "blahblah")
	under.Body = io.NopCloser(bytes.NewBufferString("some content"))

	var resp runtime.ClientResponse = response{under}
	assert.Equal(t, under.StatusCode, resp.Code())
	assert.Equal(t, under.Status, resp.Message())
	assert.Equal(t, "blahblah", resp.GetHeader("blah"))
	assert.Equal(t, []string{"blahblah"}, resp.GetHeaders("blah"))
	assert.Equal(t, under.Body, resp.Body())
}
