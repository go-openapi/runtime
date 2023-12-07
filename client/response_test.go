// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	assert.EqualValues(t, under.StatusCode, resp.Code())
	assert.Equal(t, under.Status, resp.Message())
	assert.Equal(t, "blahblah", resp.GetHeader("blah"))
	assert.Equal(t, []string{"blahblah"}, resp.GetHeaders("blah"))
	assert.Equal(t, under.Body, resp.Body())
}
