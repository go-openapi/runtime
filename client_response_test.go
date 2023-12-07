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

package runtime

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type response struct {
}

func (r response) Code() int {
	return 490
}
func (r response) Message() string {
	return "the message"
}
func (r response) GetHeader(_ string) string {
	return "the header"
}
func (r response) GetHeaders(_ string) []string {
	return []string{"the headers", "the headers2"}
}
func (r response) Body() io.ReadCloser {
	return io.NopCloser(bytes.NewBufferString("the content"))
}

func TestResponseReaderFunc(t *testing.T) {
	var actual struct {
		Header, Message, Body string
		Code                  int
	}
	reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (interface{}, error) {
		b, _ := io.ReadAll(r.Body())
		actual.Body = string(b)
		actual.Code = r.Code()
		actual.Message = r.Message()
		actual.Header = r.GetHeader("blah")
		return actual, nil
	})
	_, _ = reader.ReadResponse(response{}, nil)
	assert.Equal(t, "the content", actual.Body)
	assert.Equal(t, "the message", actual.Message)
	assert.Equal(t, "the header", actual.Header)
	assert.Equal(t, 490, actual.Code)
}

func TestResponseReaderFuncError(t *testing.T) {
	reader := ClientResponseReaderFunc(func(r ClientResponse, _ Consumer) (interface{}, error) {
		_, _ = io.ReadAll(r.Body())
		return nil, NewAPIError("fake", errors.New("writer closed"), 490)
	})
	_, err := reader.ReadResponse(response{}, nil)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "writer closed"), err.Error())

	reader = func(r ClientResponse, _ Consumer) (interface{}, error) {
		_, _ = io.ReadAll(r.Body())
		err := &fs.PathError{
			Op:   "write",
			Path: "path/to/fake",
			Err:  fs.ErrClosed,
		}
		return nil, NewAPIError("fake", err, 200)
	}
	_, err = reader.ReadResponse(response{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file already closed")

}
